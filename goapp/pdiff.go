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
	"errors"
	"image"
	"net/http"
)

type Backend interface {
	Error(r *http.Request, err error)

	// Returns a newly created site or error if it already exists.
	CreateSite(r *http.Request, name string) (*Site, error)

	// Assigns key and secret to named site.
	AssignKey(r *http.Request, name string, key, secret []byte) error

	GetSite(r *http.Request, site string) (*Site, error)
	StoreImage(r *http.Request, i image.Image, site, group string, id int64) error
	GetImageBefore(r *http.Request, site, group string, id int64) (int64, []byte, error)
	StoreDiffImage(r *http.Request, i image.Image, site, group string, id1, id2 int64, diffpx int) error
	GetUnreviewedImages(r *http.Request, site string) []DiffImage
}

type Site struct {
	Name        string
	Key, Secret []byte
}

type DiffImage struct {
	Group  string
	Id1    int64
	Id2    int64
	Pixels int
}

var (
	ErrSiteExists = errors.New("go-pdiff: site already exists")
)
