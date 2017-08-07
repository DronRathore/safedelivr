package helpers

import (
  "config"
  "models"
  "time"
  "fmt"
  "encoding/json"
  "io/ioutil"
  "strings"
  "net/http"
  Url "net/url"
)

type GithubToken struct {
  Access_Token string `json:access_token`
  Scope string `json:scope`
  Token_Type string `json:token_type`
}

var GithubClient = func () *http.Client {
  transport := &http.Transport{
    MaxIdleConns:       200,
    IdleConnTimeout:    60 * time.Second,
    DisableCompression: false,
  }
  return &http.Client{Transport: transport}
}()

func getAccessTokenRequestObject(code string, state string) *http.Request {
  var url = config.Configuration.Github.Endpoint + "/access_token"
  form := Url.Values{}
  form.Add("code", code)
  form.Add("state", state)
  form.Add("scope", "user")
  form.Add("client_id", config.Configuration.Github.ClientId)
  form.Add("client_secret", config.Configuration.Github.ClientSecret)
  form.Add("redirect_uri", config.Configuration.Github.Success)
  request, err := http.NewRequest("POST", url, strings.NewReader(form.Encode()))
  // we only cater json response
  request.Header.Set("Accept", "application/json")
  if err != nil {
    // cannot create a new Request Object
    return nil
  }
  return request
}

func getUserDataRequestObject(token string) *http.Request {
  var url = config.Configuration.Github.Api + "/user?access_token=" + token
  request, err := http.NewRequest("GET", url, nil)
  if err != nil {
    return nil
  }
  return request
}

func GetGithubUserToken(code string, state string) *GithubToken {
  var requestObject = getAccessTokenRequestObject(code, state)
  if  requestObject != nil {
    response, err := GithubClient.Do(requestObject)
    if err != nil {
      // something broken, log it
      fmt.Println(err)
      return nil
    }
    data, err := ioutil.ReadAll(response.Body)
    if err != nil {
      fmt.Println(err)
      return nil  // cannot read response body
    }
    var GithubResponse = &GithubToken{}
    err = json.Unmarshal(data, GithubResponse)
    if err != nil {
      fmt.Println(err)
      return nil // transmission broken or bad format
    }
    return GithubResponse // success, return the token struct
  }
  return nil // couldn't create request object
}

func GetGithubUserData (tokenStruct *GithubToken) *models.GithubUser {
  if tokenStruct.Access_Token != "" {
    var requestObject = getUserDataRequestObject(tokenStruct.Access_Token)
    if requestObject != nil {
      response, err := GithubClient.Do(requestObject)
      if err == nil {
        data, err := ioutil.ReadAll(response.Body)
        if err == nil {
          var GithubUserData = &models.GithubUser{Access_Token: tokenStruct.Access_Token}
          err = json.Unmarshal(data, GithubUserData)
          if err == nil {
            return GithubUserData
          }
        }
      }
    }
  }
  return nil
}