package helpers

import(
  "testing"
)

func TestIsMultipart(t *testing.T){
  var boundary = "|;|;|"
  if IsMultipart("lol; noboundary=kkk", &boundary) != false {
    t.Log("Not a multipart header")
    t.Fail()
  }
  boundary = "acdefawoeoeoeo"
  if IsMultipart("form-data; boundary=acdefawoeoeoeo", &boundary) != true {
    t.Log("Is a multipart header")
    t.Fail()
  }
  boundary = "acdefawoeoeoeo"
  if IsMultipart("form-data; boundaryies=acdefawoeoeoeo", &boundary) != false {
    t.Log("Is a multipart header")
    t.Fail()
  }
}
