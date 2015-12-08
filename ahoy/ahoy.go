package main

import (
  "os"
  "github.com/codegangsta/cli"
  "flag"
  "fmt"
  "os/exec"
  "log"
  "path"
  "path/filepath"
  "gopkg.in/yaml.v2"
  "io/ioutil"
  "sort"
  "strings"
)

type Config struct {
  Version string
  Commands map[string]Command
}

type Command struct {
  Description string
  Usage string
  Cmd string
  HideHelp bool
  SkipFlagParsing bool
}

var sourcedir string
var sourcefile string
var args []string
var verbose bool
var bashCompletion bool

func getConfigPath(sourcefile string) (string, error) {
  var err error

  // If a specific source file was set, then try to load it directly.
  if sourcefile != "" {
    if  _, err := os.Stat(sourcefile); err == nil {
      return sourcefile, err
    } else {
      fmt.Println("\n ==> Error: An ahoy config file was specified to be at", sourcefile, "but couldn't be found. Check your path.\n")
      os.Exit(1)
    }
  }

  dir, err := os.Getwd()
  if err != nil {
    log.Fatal(err)
  }
  for dir != "/" && err == nil {
    ymlpath := filepath.Join(dir, ".ahoy.yml")
    //log.Println(ymlpath)
    if _, err := os.Stat(ymlpath); err == nil {
      //log.Println("found: ", ymlpath )
      return ymlpath, err
    }
    // Chop off the last part of the path.
    dir = path.Dir(dir)
  }
  return "", err
}

func getConfig(sourcefile string) (Config, error) {

  yamlFile, err := ioutil.ReadFile(sourcefile)
  if err != nil {
    fmt.Println("\n ==> Error: An ahoy config file couldn't be found in your path. You can create an example one by using 'ahoy init'\n")
    //os.Exit(1)
  }

  var config Config

  err = yaml.Unmarshal(yamlFile, &config)
  if err != nil {
    panic(err)
  }
  return config, err
}

func getCommands(config Config) []cli.Command {
  exportCmds := []cli.Command{}

  var keys []string
  for k := range config.Commands {
      keys = append(keys, k)
  }
  sort.Strings(keys)

  for _ , name := range keys {
    cmd := config.Commands[name]
    cmdName := name
    newCmd := cli.Command{
      Name: name,
      Usage: cmd.Usage,
      SkipFlagParsing: cmd.SkipFlagParsing,
      HideHelp: cmd.HideHelp,
      Action: func(c *cli.Context) {
       args = c.Args()
       runCommand(cmdName, cmd.Cmd);
      },
    }
    //log.Println("found command: ", name, " > ", cmd.Cmd )
    exportCmds = append(exportCmds, newCmd)
  }

  return exportCmds
}

func runCommand(name string, c string) {

  cReplace := strings.Replace(c, "{{args}}", strings.Join(args, " "), 1)

  dir := sourcedir

  if verbose {
    log.Println("===> AHOY", name, "from", sourcefile, ":", cReplace)
  }
  cmd := exec.Command("bash", "-c", cReplace)
  cmd.Dir = dir
  cmd.Stdout = os.Stdout
  cmd.Stdin = os.Stdin
  cmd.Stderr = os.Stderr
  if err := cmd.Run(); err != nil {
    fmt.Fprintln(os.Stderr)
    os.Exit(1)
  }
}

func addDefaultCommands(commands []cli.Command) []cli.Command {
  newCmd := cli.Command{
    Name: "init",
    Usage: "Initialize a new .ahoy.yml config file in the current directory.",
    Action: func(c *cli.Context) {
      //log.Println(exec.LookPath(os.Args[0]))
      grabYaml := "wget https://raw.githubusercontent.com/devinci-code/ahoy/master/examples/examples.ahoy.yml -O .ahoy.yml"
      cmd := exec.Command("bash", "-c", grabYaml)
      //cmd.Dir = dir
      //cmd.Stdout = os.Stdout
      cmd.Stdin = os.Stdin
      cmd.Stderr = os.Stderr
      if err := cmd.Run(); err != nil {
        fmt.Fprintln(os.Stderr)
        os.Exit(1)
      } else {
        fmt.Println("example.ahoy.yml downloaded to the current directory. You can customize it to suit your needs!" )
      }
    },
  }

  // TODO: Check if a command has already been set. Don't add defaults if it has.
  commands = append(commands, newCmd)
  return commands
}


func init() {
  flag.StringVar(&sourcefile, "f", "", "specify the sourcefile")
  flag.BoolVar(&bashCompletion, "generate-bash-completion", false, "")
  flag.BoolVar(&verbose, "verbose", false, "")
}

// Prints the list of subcommands as the default app completion method
func BashComplete(c *cli.Context) {

  if sourcefile != "" {
    log.Println(sourcefile)
    os.Exit(0);
  }
  for _, command := range c.App.Commands {
    for _, name := range command.Names() {
      fmt.Fprintln(c.App.Writer, name)
    }
  }
}


func main() {
  // Grab the sourcefile flag first.
  flag.Parse()
  //log.Println(sourcefile)
  // cli stuff
  app := cli.NewApp()
  app.Name = "ahoy"
  app.Usage = "Send commands to docker-compose services"
  app.EnableBashCompletion = true
  app.BashComplete = BashComplete
  app.Flags = []cli.Flag {
    cli.BoolFlag{
      Name: "verbose",
      Usage: "Output extra details like the commands to be run.",
      EnvVar: "AHOY_VERBOSE",
      Destination: &verbose,
    },
    cli.StringFlag{
      Name: "f",
      Usage: "Use a specific ahoy file.",
      Destination: &sourcefile,
    },
  }
  if sourcefile, err := getConfigPath(sourcefile); err == nil {
    sourcedir = filepath.Dir(sourcefile)
    config, _ := getConfig(sourcefile)
    app.Commands = getCommands(config)
    app.Commands = addDefaultCommands(app.Commands)
    //log.Println("version: ", config.Version)
  }

  cli.AppHelpTemplate = `NAME:
   {{.Name}} - {{.Usage}}
USAGE:
   {{.HelpName}} {{if .Flags}}[global options]{{end}}{{if .Commands}} command [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}
   {{if len .Authors}}
AUTHOR(S):
   {{range .Authors}}{{ . }}{{end}}
   {{end}}{{if .Commands}}
COMMANDS:
{{range .Commands}}{{if not .HideHelp}}   {{join .Names ", "}}{{ "\t" }}{{.Usage}}{{ "\n" }}{{end}}{{end}}{{end}}{{if .Flags}}
GLOBAL OPTIONS:
   {{range .Flags}}{{.}}
   {{end}}{{end}}{{if .Copyright }}
COPYRIGHT:
   {{.Copyright}}
   {{end}}{{if .Version}}
VERSION:
   {{.Version}}
   {{end}}
`

  app.Run(os.Args)
}
