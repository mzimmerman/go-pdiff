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
	"appengine"
	"appengine/blobstore"
	"appengine/datastore"
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"net/http"

	_ "github.com/MiniProfiler/go/miniprofiler_gae"
	"github.com/mjibson/goon"
)

func init() {
	backend = &AppEngineBackend{}
}

type AppEngineBackend struct {
}

func (a *AppEngineBackend) Error(r *http.Request, err error) {
	c := appengine.NewContext(r)
	c.Errorf("go-pdiff: %v", err.Error())
}

func (a *AppEngineBackend) CreateSite(r *http.Request, name string) (*Site, error) {
	gn := goon.NewGoon(r)
	site := AESite{Id: name}
	if err := gn.RunInTransaction(func(gn *goon.Goon) error {
		if err := gn.Get(&site); err != datastore.ErrNoSuchEntity {
			return ErrSiteExists
		}
		return gn.Put(&site)
	}, nil); err != nil {
		return nil, err
	}

	return &Site{
		Name: name,
	}, nil
}

func (a *AppEngineBackend) AssignKey(r *http.Request, name string, key, secret []byte) error {
	gn := goon.NewGoon(r)
	site := AESite{Id: name}
	return gn.RunInTransaction(func(gn *goon.Goon) error {
		if err := gn.Get(&site); err != nil {
			return err
		}
		site.Key = key
		site.Secret = secret
		return gn.Put(&site)
	}, nil)
}

func (a *AppEngineBackend) GetSite(r *http.Request, name string) (*Site, error) {
	gn := goon.NewGoon(r)
	site := &AESite{Id: name}
	err := gn.Get(site)
	return site.toSite(), err
}

func (a *AESite) toSite() *Site {
	return &Site{
		Name:   a.Id,
		Key:    a.Key,
		Secret: a.Secret,
	}
}

func (a AppEngineBackend) StoreImage(r *http.Request, i image.Image, name, group string, id int64) error {
	c := appengine.NewContext(r)
	gn := goon.NewGoon(r)
	site := &AESite{Id: name}
	aegroup := &AEGroup{Id: group, Site: gn.Key(site)}
	im := &AEImage{
		Id:     id,
		Group:  gn.Key(aegroup),
		Width:  i.Bounds().Size().X,
		Height: i.Bounds().Size().Y,
	}

	w, err := blobstore.Create(c, "image/png")
	if err != nil {
		return err
	}
	png.Encode(w, i)
	if err = w.Close(); err != nil {
		return err
	}
	bk, err := w.Key()
	if err != nil {
		return err
	}
	im.Blob = bk
	return gn.PutMany(im, aegroup)
}

func (a AppEngineBackend) GetImageBefore(r *http.Request, name, group string, id int64) (int64, []byte, error) {
	gn := goon.NewGoon(r)
	c := appengine.NewContext(r)
	site := &AESite{Id: name}
	aegroup := &AEGroup{Id: group, Site: gn.Key(site)}
	q := datastore.NewQuery(gn.Key(&AEImage{}).Kind()).KeysOnly()
	q = q.Ancestor(gn.Key(aegroup))
	q = q.Order("-__key__")
	keys, err := gn.GetAll(q.Offset(1).Limit(1), nil)
	if err != nil || len(keys) != 1 {
		return 0, nil, err
	}
	i := AEImage{
		Id:    keys[0].IntID(),
		Group: keys[0].Parent(),
	}
	if err := gn.Get(&i); err != nil {
		return 0, nil, err
	}
	br := blobstore.NewReader(c, i.Blob)
	ib, err := ioutil.ReadAll(br)
	return i.Id, ib, err
}

func (a AppEngineBackend) StoreDiffImage(r *http.Request, i image.Image, name, group string, id1, id2 int64, diffpx int) error {
	gn := goon.NewGoon(r)
	c := appengine.NewContext(r)
	site := &AESite{Id: name}
	aegroup := &AEGroup{Id: group, Site: gn.Key(site)}
	aedi := &AEDiffImage{
		Id:     fmt.Sprintf("%v,%v", id1, id2),
		Group:  gn.Key(aegroup),
		Id1:    id1,
		Id2:    id2,
		Pixels: diffpx,
	}

	if i != nil {
		w, err := blobstore.Create(c, "image/png")
		if err != nil {
			return err
		}
		png.Encode(w, i)
		if err = w.Close(); err != nil {
			return err
		}
		bk, err := w.Key()
		if err != nil {
			return err
		}
		aedi.Blob = bk
	}

	return gn.Put(aedi)
}

func (a AppEngineBackend) GetUnreviewedImages(r *http.Request, name string) []DiffImage {
	gn := goon.NewGoon(r)
	site := &AESite{Id: name}
	if err := gn.Get(site); err != nil {
		return nil
	}

	q := datastore.NewQuery(gn.Key(&AEDiffImage{}).Kind())
	q = q.Ancestor(gn.Key(site))
	q = q.Filter("r", false)
	q = q.KeysOnly()
	var aedis []*AEDiffImage
	gn.GetAll(q, &aedis)
	dis := make([]DiffImage, len(aedis))
	for i, v := range aedis {
		dis[i] = DiffImage{
			Group:  v.Group.StringID(),
			Id1:    v.Id1,
			Id2:    v.Id2,
			Pixels: v.Pixels,
		}
	}
	return dis
}

type AESite struct {
	_kind  string `goon:"kind,S"`
	Id     string `datastore:"-" goon:"id"`
	Key    []byte `datastore:"k,noindex"`
	Secret []byte `datastore:"s,noindex"`
}

type AEImage struct {
	_kind  string            `goon:"kind,I"`
	Id     int64             `datastore:"-" goon:"id"`
	Group  *datastore.Key    `datastore:"-" goon:"parent"`
	Width  int               `datastore:"w,noindex"`
	Height int               `datastore:"h,noindex"`
	Blob   appengine.BlobKey `datastore:"b,noindex"`
}

type AEDiffImage struct {
	_kind    string         `goon:"kind,D"`
	Id       string         `datastore:"-" goon:"id"`
	Group    *datastore.Key `datastore:"-" goon:"parent"`
	Id1      int64
	Id2      int64
	Reviewed bool              `datastore:"r"`
	Pixels   int               `datastore:"p,noindex"`
	Blob     appengine.BlobKey `datastore:"b,noindex"`
}

type AEGroup struct {
	_kind string         `goon:"kind,G"`
	Id    string         `datastore:"-" goon:"id"`
	Site  *datastore.Key `datastore:"-" goon:"parent"`
}
