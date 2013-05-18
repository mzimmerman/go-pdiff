/*
 * Copyright (c) 2013 Matt Jibson <matt.jibson@gmail.com>
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package goapp

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MiniProfiler/go/miniprofiler"
	"github.com/gorilla/mux"
	"html/template"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

var router = new(mux.Router)
var templates *template.Template
var backend Backend

func init() {
	var err error

	if templates, err = template.New("").Funcs(funcs).
		ParseFiles(
		"templates/base.html",
	); err != nil {
		log.Fatal(err)
	}

	router.Handle("/", miniprofiler.NewHandler(Main)).Name("main")
	router.Handle("/create-site", miniprofiler.NewHandler(CreateSite)).Name("create-site")
	router.Handle("/post-image", miniprofiler.NewHandler(PostImage)).Name("post-image")

	http.Handle("/", router)

	miniprofiler.Position = "right"
	miniprofiler.ShowControls = false
}

func serveError(w http.ResponseWriter, r *http.Request, err error) {
	backend.Error(r, err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func Main(p *miniprofiler.Profile, w http.ResponseWriter, r *http.Request) {
	if err := templates.ExecuteTemplate(w, "base.html", includes(p, r)); err != nil {
		serveError(w, r, err)
	}
}

func CreateSite(p *miniprofiler.Profile, w http.ResponseWriter, r *http.Request) {
	s := strings.TrimSpace(r.FormValue("site"))
	if len(s) == 0 {
		serveError(w, r, errors.New("no site"))
		return
	}

	site, err := backend.CreateSite(r, s)
	if err != nil {
		serveError(w, r, err)
		return
	}

	var ok bool
	site.Key, site.Secret, ok = createKeySecret()
	if !ok {
		serveError(w, r, errors.New("couldn't create key"))
		return
	}
	backend.AssignKey(r, s, site.Key, site.Secret)
	b, _ := json.Marshal(&site)
	w.Write(b)
}

// final bool is ok: false = error, true = success
func createKeySecret() ([]byte, []byte, bool) {
	c := 64
	b := make([]byte, c)
	n, err := io.ReadFull(rand.Reader, b)
	if n != len(b) || err != nil {
		return nil, nil, false
	}
	return b[:c/2], b[c/2:], true
}

/*
PostImage uploads an image.

Request is a POST, where the body is a json object with the following key/value pairs:
key: (non-secret) base-64 encoded key
site (string): name of the site
group (string): image group name (generally a URL)
id (int): image id number (generally build number or timestamp). images are sorted by this (not submission time).
image: standard base64-encoded image data. any go-supported image format is supported.
signature: SHA256(HMAC(secret, base64-encoded key + site + group + id + base64-encoded image)). use the base-10 string version of id, and concatenate all values into one string. hmac then sha256 it. \n should be inserted between each element of the hmac value.

if there is an existing image with same group and id, an error is returned.
*/
func PostImage(p *miniprofiler.Profile, w http.ResponseWriter, r *http.Request) {
	post := Post{}
	body, _ := ioutil.ReadAll(r.Body)
	if err := json.Unmarshal(body, &post); err != nil {
		serveError(w, r, err)
		return
	}

	site, err := backend.GetSite(r, post.Site)
	if err != nil {
		serveError(w, r, err)
		return
	}

	sigin := fmt.Sprintf("%x\n%v\n%v\n%v\n%x", site.Key, site.Name, post.Group, post.Id, post.Image)
	h := hmac.New(sha256.New, site.Secret)
	h.Write([]byte(sigin))
	signature := fmt.Sprintf("%x", h.Sum(nil))

	if !bytes.Equal(site.Key, post.Key) {
		serveError(w, r, errors.New("go-pdiff: keys did not match"))
		return
	}
	if signature != post.Signature {
		serveError(w, r, errors.New("go-pdiff: signatures did not match"))
		return
	}

	im, _, err := image.Decode(bytes.NewBuffer(post.Image))
	if err != nil {
		serveError(w, r, err)
		return
	}

	if err = backend.StoreImage(r, im, site.Name, post.Group, post.Id); err != nil {
		serveError(w, r, err)
		return
	}
}

type Post struct {
	Key       []byte `json:"key"`
	Site      string `json:"site"`
	Group     string `json:"group"`
	Id        int64  `json:"id"`
	Image     []byte `json:"image"`
	Signature string `json:"signature"`
}
