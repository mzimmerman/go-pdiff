package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

func main() {
	resp, _ := http.PostForm("http://localhost:8080/create-site",
		url.Values{"site": {time.Now().String()}})
	site := struct {
		Name        string
		Key, Secret []byte
	}{}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, &site)
	fmt.Println(site)
	imbytes, _ := ioutil.ReadFile("im.png")
	p := Post{
		Key:   site.Key,
		Site:  site.Name,
		Group: "group",
		Id:    time.Now().UnixNano(),
		Image: imbytes,
	}

	sigin := fmt.Sprintf("%x\n%v\n%v\n%v\n%x", site.Key, site.Name, p.Group, p.Id, p.Image)
	h := hmac.New(sha256.New, site.Secret)
	h.Write([]byte(sigin))
	p.Signature = fmt.Sprintf("%x", h.Sum(nil))
	j, _ := json.MarshalIndent(&p, "", "\t")
	fmt.Println(string(j))
	buf := bytes.NewBuffer(j)
	resp, _ = http.Post("http://localhost:8080/post-image", "text/json", buf)
}

type Post struct {
	Key       []byte `json:"key"`
	Site      string `json:"site"`
	Group     string `json:"group"`
	Id        int64  `json:"id"`
	Image     []byte `json:"image"`
	Signature string `json:"signature"`
}

type T struct {
	B []byte
}
