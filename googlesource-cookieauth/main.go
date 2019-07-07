// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Googlesource-cookieauth is a command that writes Netscape cookie file for
// googlesource.com / source.developers.google.com.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/aki237/nscjar"
	"github.com/google/googlesource-auth-tools/credentials"
)

var (
	outputFile string
)

func init() {
	const (
		usage = "the output filepath. If unspecified, defaults to $HOME/.git-credential-cache/googlesource-cookieauth-cookie"
	)
	flag.StringVar(&outputFile, "output", "", usage)
	flag.StringVar(&outputFile, "o", "", usage)
}

func main() {
	flag.Parse()
	gitBinary, err := credentials.FindGitBinary()
	if err != nil {
		log.Fatalf("Cannot find the git binary: %v", err)
	}
	urls, err := gitBinary.ListURLs(context.Background())
	if err != nil {
		log.Fatalf("Cannot read the list of URLs in git-config: %v", err)
	}
	var hasGoogleSource, hasSourceDevelopers bool
	for _, u := range urls {
		if u.Host == "googlesource.com" && (u.Path == "" || u.Path == "/") {
			hasGoogleSource = true
		}
		if u.Host == "source.developers.google.com" && (u.Path == "" || u.Path == "/") {
			hasSourceDevelopers = true
		}
	}
	if !hasGoogleSource {
		urls = append(urls, &url.URL{Scheme: "https", Host: "googlesource.com"})
	}
	if !hasSourceDevelopers {
		urls = append(urls, &url.URL{Scheme: "https", Host: "source.developers.google.com"})
	}

	cookies := []*http.Cookie{}
	for _, u := range urls {
		token, err := credentials.MakeToken(context.Background(), u)
		if err != nil {
			log.Fatalf("Cannot create a token for %s: %v", u, err)
		}
		cookies = append(cookies, credentials.MakeCookies(u, token)...)
	}

	if outputFile == "" {
		outputFile, err = gitBinary.PathConfig(context.Background(), "google.cookieFile")
		if err != nil {
			log.Fatalf("Cannot read google.cookieFile in git-config: %v", err)
		}
	}
	if outputFile == "" {
		u, err := user.Current()
		if err != nil {
			log.Fatalf("Cannot get the current user: %v", err)
		}
		outputFile = filepath.Join(u.HomeDir, ".git-credential-cache", "googlesource-cookieauth-cookie")
	}

	var w io.Writer
	if outputFile == "-" {
		w = os.Stdout
	} else {
		if err := os.MkdirAll(filepath.Dir(outputFile), 0700); err != nil {
			log.Fatalf("Cannot create the output directory: %v", err)
		}
		w, err = os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
		if err != nil {
			log.Fatalf("Cannot open the output file: %v", err)
		}
	}

	fmt.Fprintf(w, "# Created by %s at %s\n", os.Args[0], time.Now().Format(time.RFC3339))
	p := nscjar.Parser{}
	for _, c := range cookies {
		p.Marshal(w, c)
	}
}