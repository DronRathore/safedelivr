package config

import (
  "os"
  "fmt"
  "io/ioutil"
  "gopkg.in/yaml.v2"
)

type Config struct {
  cwd string
  Port string `yaml:port`
  Redis struct {
    Ip string `yaml:ip`
    Port string `yaml:port`
    Db int `yaml:db`
  }
  Session struct {
    Cookie string `yaml:cookie`
    Key string `yaml:key`
  }
  Sendgrid struct {
    Key string `yaml:key`
  }
  Cassandra struct {
    Ip string `yaml:ip`
    Keyspace string `yaml:keyspace`
  }
  MailGun struct {
    Api_Key string `yaml:key`
    Pub_Key string `yaml:pubkey`
  }
  Github struct {
    ClientId string `yaml:clientid`
    ClientSecret string `yaml:clientsecret`
    Endpoint string `yaml:endpoint`
    Success string `yaml:success`
    Api string `yaml:api`
  }
}
// global configuration var
var Configuration *Config

func (c *Config) SetCwd(cwd string){
  c.cwd = cwd
}
func (c *Config) GetCwd() string {
  return c.cwd
}
func Init(){
  Configuration = &Config{}
  var cwd, err = os.Getwd()
  if err != nil {
    panic(err)
  }
  Configuration.SetCwd(cwd)
  LoadYaml()
}

func LoadYaml(){
  file, err := os.Open(Configuration.GetCwd() + "/application.yml")
  if err != nil {
    fmt.Printf("Cannot load application.yml from root")
    panic(err)
  }
  data, err := ioutil.ReadAll(file)
  if err != nil {
    fmt.Printf("Cannot read application.yml file, check access.")
    panic(err)
  }
  err = yaml.Unmarshal(data, Configuration)
  if err != nil {
    fmt.Printf("Yaml parse failed")
    panic(err)
  }
}