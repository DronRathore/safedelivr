package models

// these data points can be extended further
// that's why kept seperate from UserRedis struct
type GithubUser struct{
  UUID string `json:uuid`
  Avatar_Url string `json:avatar_url`
  Name string `json:name`
  Company string `json:company`
  Location string `json:location`
  Email string `json:email`
  Access_Token string `json:access_token`
}