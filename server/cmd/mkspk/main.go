// mkspk assemble le paquet Synology AURORA.spk de façon reproductible : entrées triées, dates fixes,
// propriétaire root, aucun attribut étendu macOS (._*), permissions explicites.
//
// Structure produite (guide développeur Synology, DSM 7) :
//
//	INFO, LICENSE, PACKAGE_ICON.PNG, PACKAGE_ICON_256.PNG, conf/{privilege,resource}, scripts/*, package.tgz
//	package.tgz : bin/{aurora,ffmpeg,ffprobe}, ui/{config,images/*}, port_conf/aurora.sc, licenses/*
package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type entry struct {
	name string
	mode int64
	data []byte
	dir  bool
}

func main() {
	var (
		version = flag.String("version", "", "version (ex. 0.1.0)")
		build   = flag.String("build", "0001", "numéro de build DSM (à incrémenter à chaque publication)")
		bin     = flag.String("bin", "", "binaire aurora linux/amd64")
		ffdir   = flag.String("ffmpeg", "", "dossier contenant ffmpeg, ffprobe et GPLv3.txt (statiques linux/amd64)")
		pkgdir  = flag.String("pkg", "packaging/synology", "dossier de packaging")
		out     = flag.String("out", "", "fichier .spk à produire")
		epoch   = flag.Int64("mtime", 0, "date fixe des entrées (Unix) ; défaut SOURCE_DATE_EPOCH ou 0")
	)
	flag.Parse()
	if *version == "" || *bin == "" || *ffdir == "" || *out == "" {
		flag.Usage()
		os.Exit(2)
	}
	mtime := time.Unix(*epoch, 0)
	if *epoch == 0 {
		if v := os.Getenv("SOURCE_DATE_EPOCH"); v != "" {
			var e int64
			fmt.Sscan(v, &e)
			mtime = time.Unix(e, 0)
		} else {
			mtime = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		}
	}
	if err := run(*version, *build, *bin, *ffdir, *pkgdir, *out, mtime); err != nil {
		fmt.Fprintln(os.Stderr, "mkspk :", err)
		os.Exit(1)
	}
}

func read(path string) []byte {
	b, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "mkspk :", err)
		os.Exit(1)
	}
	return b
}

func run(version, build, bin, ffdir, pkgdir, out string, mtime time.Time) error {
	// 1. Contenu installé (package.tgz).
	inner := []entry{
		{name: "bin", dir: true, mode: 0o755},
		{name: "bin/aurora", mode: 0o755, data: read(bin)},
		{name: "bin/ffmpeg", mode: 0o755, data: read(filepath.Join(ffdir, "ffmpeg"))},
		{name: "bin/ffprobe", mode: 0o755, data: read(filepath.Join(ffdir, "ffprobe"))},
		{name: "licenses", dir: true, mode: 0o755},
		{name: "licenses/FFMPEG-GPLv3.txt", mode: 0o644, data: read(filepath.Join(ffdir, "GPLv3.txt"))},
		{name: "licenses/FFMPEG-BUILD-README.txt", mode: 0o644, data: read(filepath.Join(ffdir, "readme.txt"))},
		{name: "licenses/THIRD_PARTY_NOTICES.md", mode: 0o644, data: read(filepath.Join(pkgdir, "THIRD_PARTY_NOTICES.md"))},
		{name: "ui", dir: true, mode: 0o755},
		{name: "ui/config", mode: 0o644, data: read(filepath.Join(pkgdir, "package/ui/config"))},
		{name: "ui/images", dir: true, mode: 0o755},
		{name: "port_conf", dir: true, mode: 0o755},
		{name: "port_conf/aurora.sc", mode: 0o644, data: read(filepath.Join(pkgdir, "package/port_conf/aurora.sc"))},
	}
	for _, s := range []int{16, 24, 32, 48, 64, 72, 256} {
		n := fmt.Sprintf("aurora_%d.png", s)
		inner = append(inner, entry{name: "ui/images/" + n, mode: 0o644, data: read(filepath.Join(pkgdir, "icons", n))})
	}
	var tgz bytes.Buffer
	gz, _ := gzip.NewWriterLevel(&tgz, gzip.BestCompression)
	if err := writeTar(gz, inner, mtime); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	var size int64
	for _, e := range inner {
		size += int64(len(e.data))
	}
	sum := md5.Sum(tgz.Bytes())

	// 2. INFO.
	info := string(read(filepath.Join(pkgdir, "INFO.in")))
	info = strings.NewReplacer("@VERSION@", version, "@BUILD@", build, "@CHECKSUM@", hex.EncodeToString(sum[:]),
		"@EXTRACTSIZE@", fmt.Sprint(size/1024+1)).Replace(info)

	// 3. Paquet externe.
	outer := []entry{
		{name: "INFO", mode: 0o644, data: []byte(info)},
		{name: "LICENSE", mode: 0o644, data: read(filepath.Join(pkgdir, "LICENSE"))},
		{name: "PACKAGE_ICON.PNG", mode: 0o644, data: read(filepath.Join(pkgdir, "icons/PACKAGE_ICON.PNG"))},
		{name: "PACKAGE_ICON_256.PNG", mode: 0o644, data: read(filepath.Join(pkgdir, "icons/PACKAGE_ICON_256.PNG"))},
		{name: "conf", dir: true, mode: 0o755},
		{name: "conf/privilege", mode: 0o644, data: read(filepath.Join(pkgdir, "conf/privilege"))},
		{name: "conf/resource", mode: 0o644, data: read(filepath.Join(pkgdir, "conf/resource"))},
		{name: "scripts", dir: true, mode: 0o755},
		{name: "package.tgz", mode: 0o644, data: tgz.Bytes()},
	}
	err := filepath.WalkDir(filepath.Join(pkgdir, "scripts"), func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || strings.HasPrefix(d.Name(), ".") {
			return err
		}
		outer = append(outer, entry{name: "scripts/" + d.Name(), mode: 0o755, data: read(p)})
		return nil
	})
	if err != nil {
		return err
	}
	var spk bytes.Buffer
	if err := writeTar(&spk, outer, mtime); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(out, spk.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Printf("%s : %.1f Mo (package.tgz %.1f Mo, %.1f Mo installés)\n", out,
		float64(spk.Len())/(1<<20), float64(tgz.Len())/(1<<20), float64(size)/(1<<20))
	return nil
}

type writerTo interface{ Write([]byte) (int, error) }

func writeTar(w writerTo, entries []entry, mtime time.Time) error {
	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })
	tw := tar.NewWriter(w)
	for _, e := range entries {
		h := &tar.Header{Name: e.name, Mode: e.mode, ModTime: mtime, Uid: 0, Gid: 0, Uname: "root", Gname: "root",
			Format: tar.FormatUSTAR}
		if e.dir {
			h.Typeflag = tar.TypeDir
			h.Name += "/"
		} else {
			h.Typeflag = tar.TypeReg
			h.Size = int64(len(e.data))
		}
		if err := tw.WriteHeader(h); err != nil {
			return fmt.Errorf("%s : %w", e.name, err)
		}
		if !e.dir {
			if _, err := tw.Write(e.data); err != nil {
				return err
			}
		}
	}
	return tw.Close()
}
