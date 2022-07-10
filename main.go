package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/russross/blackfriday/v2"
)

var (
	listen = flag.String("l", ":8080", "set http listener to [ip]:port")
	root   = flag.String("root", "root", "werc webroot")

	indexFiles = []string{"index", "README"}
)

type WercConfig struct {
	MasterSite string
	Title      string
	Subtitle   string
}

type MenuEntry struct {
	Name string
	Path string
	This bool
	Sub  []*MenuEntry
}

type MenuEntries []*MenuEntry

func (me MenuEntries) Len() int           { return len(me) }
func (me MenuEntries) Swap(i, j int)      { me[i], me[j] = me[j], me[i] }
func (me MenuEntries) Less(i, j int) bool { return me[i].Name < me[j].Name }

type WercPage struct {
	Title   string        // <head> title
	Menu    MenuEntries   // menu entries
	Content template.HTML // inner page content
	Config  WercConfig    // site-specific config
}

type Werc struct {
	root string
	conf WercConfig
	tmpl *template.Template

	fs *FS
}

func New(root string) *Werc {
	w := new(Werc)
	w.root = root
	w.tmpl = template.New("root")

	var err error

	w.fs, err = NewFS(root)
	if err != nil {
		log.Printf("can't open root: %v", err)
		return nil
	}

	b, err := readfile(w.fs, "etc/config.json")
	if err != nil {
		log.Printf("error loading config.json: %v", err)
		return nil
	}

	err = json.Unmarshal(b, &w.conf)
	if err != nil {
		log.Printf("%s: %s", root+"/config.json", err)
		return nil
	}

	// load templates
	tmpls := []string{"base", "directory", "footer", "menu", "text", "topbar"}
	for _, tn := range tmpls {
		b, err := readfile(w.fs, fmt.Sprintf("lib/%s.html", tn))
		if err != nil {
			panic(err)
		}
		template.Must(w.tmpl.New(tn + ".html").Parse(string(b)))
	}

	return w
}

func cleanname(s string) string {
	if s == "_werc" {
		return ""
	}

	if strings.HasPrefix(s, "index") {
		return ""
	}

	switch s {
	case "sitemap.txt", "sitemap.gz":
		return ""
	}

	for _, suf := range []string{".md", ".txt", ".html"} {
		if strings.HasSuffix(s, suf) {
			return strings.TrimSuffix(s, suf)
		}
	}
	return s
}

func ptitle(s string) string {
	s = strings.TrimSuffix(s, "/")
	if idx := strings.LastIndex(s, "index"); idx != -1 {
		s = s[:idx-1]
	}
	_, file := filepath.Split(s)
	for _, suf := range []string{".md", ".txt", ".html"} {
		if strings.HasSuffix(file, suf) {
			return strings.TrimSuffix(file, suf)
		}
	}
	return file
}

func (werc *Werc) genmenu(site, dir string) MenuEntries {
	var dirs []string
	var root MenuEntries

	base := "sites/" + site

	spl := strings.Split(strings.TrimPrefix(filepath.Clean(dir), "/"), string(filepath.Separator))

	_, current := filepath.Split(dir)

	if current != "" {
		spl = spl[:len(spl)-1]
	}

	//log.Printf("base %s path %s spl %+v", base, dir, spl)

	dirs = append(dirs, "/")

	for i := range spl {
		path := "/" + filepath.Join(spl[:i+1]...)
		dirs = append(dirs, path)
	}

	//log.Printf("dirs %v", dirs)

	var last MenuEntries
	for i := range dirs {
		var sub MenuEntries
		b := filepath.Join(base, dirs[i])
		fi, _ := readdir(werc.fs, b)
		for _, f := range fi {
			newname, ok := okmenu(b, f)
			if !ok {
				continue
			}
			me := &MenuEntry{Name: newname, Path: filepath.Join(dirs[i], newname)}
			if f.Mode().IsDir() {
				me.Path = me.Path + "/"
				me.Name = me.Name + "/"
			}
			// if browing a file, mark it as current
			if me.Name == current {
				me.This = true
			}
			//log.Printf("me %+v", me)
			sub = append(sub, me)
		}

		if sub != nil {
			sort.Sort(sub)
		}

		if dirs[i] == "/" {
			root = sub
			last = root
		} else {
			// mark directories as currently being browsed
			for l, v := range last {
				_, file := filepath.Split(dirs[i])
				if v.Name == file+"/" {
					last[l].This = true
					last[l].Sub = sub
					last = sub
				}
			}
		}
	}

	sort.Sort(root)

	return root
}

func (werc *Werc) WercCommon(w http.ResponseWriter, r *http.Request, site string, page *WercPage) {
	path := r.URL.Path

	page.Menu = werc.genmenu(site, path)

	conf := "sites/" + site + "/_werc/config.json"
	b, err := readfile(werc.fs, conf)
	if err != nil {
		log.Printf("%s: %s", conf, err)
	} else {
		err = json.Unmarshal(b, &page.Config)
		if err != nil {
			log.Printf("%s: %s", conf, err)
		}
	}
	//log.Printf("root %+v", page.Menu)
	if err := werc.tmpl.ExecuteTemplate(w, "base.html", page); err != nil {
		log.Printf("%s: %s", r.URL, err)
	}
}

// returns true if a path name is ok to show in the navigation
func okmenu(base string, fi os.FileInfo) (string, bool) {
	if strings.HasPrefix(fi.Name(), ".") {
		return "", false
	}
	if fi.Name() == "_werc" {
		return "", false
	}
	for _, index := range indexFiles {
		if strings.HasPrefix(fi.Name(), index+".") {
			return "", false
		}
	}
	if strings.Contains(fi.Name(), "sitemap.") {
		return "", false
	}
	if fi.Mode().IsDir() {
		return fi.Name(), true
	}
	for _, suf := range []string{".md", ".txt", ".html"} {
		if strings.HasSuffix(fi.Name(), suf) {
			return strings.TrimSuffix(fi.Name(), suf), true
		}
	}
	return "", false
}

func (werc *Werc) WercDir(w http.ResponseWriter, r *http.Request, site, dir string) {
	type DirEntry struct {
		Name string
		Fi   os.FileInfo
	}

	type DirData struct {
		Title   string
		Entries []DirEntry
	}

	var data DirData
	data.Title = r.URL.Path

	buf := new(bytes.Buffer)
	fi, err := readdir(werc.fs, dir)
	if err != nil {
		http.Error(w, fmt.Sprintf("%s", err), 500)
		return
	}
	for _, f := range fi {
		if name := cleanname(f.Name()); name != "" {
			e := DirEntry{Name: name, Fi: f}
			data.Entries = append(data.Entries, e)
		}
	}

	werc.tmpl.ExecuteTemplate(buf, "directory.html", data)
	werc.WercCommon(w, r, site, &WercPage{Title: ptitle(dir), Content: template.HTML(buf.String())})
}

func (werc *Werc) WercMd(w http.ResponseWriter, r *http.Request, site, path string) {
	b, err := readfile(werc.fs, path)
	if err != nil {
		http.Error(w, fmt.Sprintf("%s", err), 404)
		return
	}
	md := blackfriday.Run(b)
	werc.WercCommon(w, r, site, &WercPage{Title: ptitle(path), Content: template.HTML(string(md))})
}

func (werc *Werc) WercHTML(w http.ResponseWriter, r *http.Request, site, path string) {
	b, err := readfile(werc.fs, path)
	if err != nil {
		http.Error(w, fmt.Sprintf("%s", err), 404)
		return
	}
	werc.WercCommon(w, r, site, &WercPage{Title: ptitle(path), Content: template.HTML(string(b))})
}

func (werc *Werc) WercTXT(w http.ResponseWriter, r *http.Request, site, path string) {
	b, err := readfile(werc.fs, path)
	if err != nil {
		http.Error(w, fmt.Sprintf("%s", err), 404)
		return
	}

	buf := new(bytes.Buffer)
	werc.tmpl.ExecuteTemplate(buf, "text.html", string(b))
	werc.WercCommon(w, r, site, &WercPage{Title: ptitle(path), Content: template.HTML(buf.String())})
}

func (werc *Werc) Pub(w http.ResponseWriter, r *http.Request, path string) {
	if strings.HasPrefix(path, "/") {
		path = path[1:]
	}

	b, err := readfile(werc.fs, path)
	if err != nil {
		log.Printf("Pub: %v", err)
		http.Error(w, err.Error(), 404)
		return
	}

	buf := bytes.NewReader(b)
	http.ServeContent(w, r, filepath.Base(path), time.Now(), buf)

	log.Printf("pub sent %d bytes", len(b))
}

func (werc *Werc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s", r.URL)
	site, _, _ := net.SplitHostPort(r.Host)
	if site == "" {
		site = werc.conf.MasterSite
	}
	path := r.URL.Path

	// try pub first
	if strings.HasPrefix(path, "/pub") {
		werc.Pub(w, r, path)
		return
	}

again:
	base := "sites/" + site

	if strings.HasSuffix(path, "/index") {
		http.Redirect(w, r, strings.TrimSuffix(path, "/index"), http.StatusMovedPermanently)
		return
	}

	if !strings.HasSuffix(path, "/") {
		f, err := werc.fs.Open(base + path)
		if err == nil {
			defer f.Close()
			fi, err := f.Stat()
			if err != nil && fi.IsDir() {
				http.Redirect(w, r, path+"/", http.StatusMovedPermanently)
				return
			}
		}
	}

	// various format handling
	sufferring := map[string]func(w http.ResponseWriter, r *http.Request, site, path string){
		"md":   werc.WercMd,
		"html": werc.WercHTML,
		"txt":  werc.WercTXT,
	}

	log.Printf("path %v", base)

	for suf, handler := range sufferring {
		var tryfiles []string
		if strings.HasSuffix(path, "/") {
			for _, index := range indexFiles {
				tryfiles = append(tryfiles, filepath.Join(base, path, index+"."+suf))
			}
		} else {
			tryfiles = append(tryfiles, filepath.Join(base, path+"."+suf))
		}

		for _, f := range tryfiles {
			fh, err := werc.fs.Open(f)
			if err != nil {
				continue
			}

			defer fh.Close()

			log.Printf("%s %s", suf, f)
			handler(w, r, site, f)
			return
		}
	}

	if f, err := werc.fs.Open(base + path); err == nil {
		defer f.Close()

		st, _ := f.Stat()
		if st.Mode().IsDir() {
			// directory handling
			log.Printf("d %s", base+path)
			werc.WercDir(w, r, site, base+path)
			return
		}

		// plain file handling
		log.Printf("f %s", base+path)

		// ripped from http.serveContent
		ctype := mime.TypeByExtension(filepath.Ext(path))
		if ctype == "" {
			// read a chunk to decide between utf-8 text and binary
			var buf [512]byte
			n, _ := io.ReadFull(f, buf[:])
			ctype = http.DetectContentType(buf[:n])
			_, err := f.Seek(0, os.SEEK_SET) // rewind to output whole file
			if err != nil {
				http.Error(w, "seeker can't seek", http.StatusInternalServerError)
				return
			}
		}
		w.Header().Set("Content-Type", ctype)

		io.Copy(w, f)
		return
	}

	if site != werc.conf.MasterSite {
		site = werc.conf.MasterSite
		goto again
	}

	log.Printf("404 %s", path)

	http.NotFound(w, r)
}

func main() {
	flag.Parse()

	w := New(*root)
	if w == nil {
		log.Fatal("can't create root")
	}

	mux := http.NewServeMux()
	mux.Handle("/", w)

	var listener net.Listener
	var tlsconf *tls.Config
	var err error

	listener, err = net.Listen("tcp", *listen)

	if err != nil {
		log.Fatal(err)
	}

	s := &http.Server{
		Addr:           *listen,
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
		TLSConfig:      tlsconf,
	}

	log.Fatal(s.Serve(listener))
}
