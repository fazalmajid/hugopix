package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/artyom/smartcrop"
	"github.com/nfnt/resize"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/termie/go-shutil"
)

const iso_8601 = "2006-01-02 15:04:05"

type Pix struct {
	Filename, Small, Thumbnail             string
	Width, Height, SmallWidth, SmallHeight int
	Copyright                              string
}

var (
	verbose int
	pix     []Pix
	small_w uint
	small_h uint
	thm_w   uint
	thm_h   uint
	target  string
)

func thumbnail(path string, fn_stat os.FileInfo) error {
	fn := filepath.Join(target, filepath.Base(path))
	if verbose > 1 {
		log.Println("fn", fn)
	}
	
	if strings.Contains(fn, "_thm.") || strings.Contains(fn, "_small.") {
		if verbose > 1 {
			log.Println("_thm or _small", path)
		}
		return nil
	}
	ext_offset := strings.LastIndex(fn, ".")
	if ext_offset == -1 {
		if verbose > 1 {
			log.Println("no extension", path)
		}
		return nil
	}
	ext := fn[ext_offset:]
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg", ".png":
		break
	default:
		if verbose > 1 {
			log.Println("wrong extension", path)
		}
		return nil
	}
	_, err := os.Stat(fn)
	if err == nil {
		os.Remove(fn)
	}
	err = os.Link(path, fn)
	if err != nil {
		err = shutil.CopyFile(path, fn, false)
		if err != nil {
			return err
		}
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	img, format, err := image.Decode(f)
	if err != nil {
		return err
	}
	f.Seek(0, 0)
	bounds := img.Bounds().Size()
	meta, err := exif.Decode(f)
	copyright := ""
	if err == nil {
		copyright_tag, err := meta.Get(exif.Copyright)
		if err == nil {
			copyright = copyright_tag.String()
		}
	}
	copyright = strings.Trim(copyright, "\"")
	small_fn := strings.Replace(fn, ext, "_small"+ext, -1)
	thm_fn := strings.Replace(fn, ext, "_thm"+ext, -1)
	small_stat, err := os.Stat(small_fn)
	small_img := resize.Thumbnail(small_w, small_h, img, resize.Lanczos2)
	sbounds := small_img.Bounds().Size()
	pix = append(pix, Pix{filepath.Base(fn), filepath.Base(small_fn), filepath.Base(thm_fn), bounds.X, bounds.Y, sbounds.X, sbounds.Y, copyright})
	if err != nil || small_stat.ModTime().Before(fn_stat.ModTime()) {
		// regenerate the small if it is more older than the image
		if verbose > 0 {
			log.Println("generating small", small_fn, "for", format, fn)
		}
		small_f, err := os.Create(small_fn)
		defer small_f.Close()
		if err != nil {
			return err
		}
		switch format {
		case "jpeg":
			jpeg.Encode(small_f, small_img, nil)
		case "png":
			png.Encode(small_f, small_img)
		default:
			return errors.New("unexpected format: " + format)
		}
	}
	thm_stat, err := os.Stat(thm_fn)
	if err == nil && thm_stat.ModTime().After(fn_stat.ModTime()) {
		// do not regenerate the thumbnail if it is more recent than the image
		if verbose > 1 {
			log.Println("thumbnail more recent than original", path)
		}
		return nil
	}
	// if verbose > 0 {
	// 	log.Println("generating thumbnail", thm_fn, "for", fn, "format", format, "metadata", meta)
	// }
	crop, err := smartcrop.Crop(img, int(thm_w), int(thm_h))
	if err != nil {
		return err
	}
	if verbose > 0 {
		log.Println("\tthe best crop is", crop)
	}
	thm_sub, ok := img.(interface {
		SubImage(r image.Rectangle) image.Image
	})
	if !ok {
		if verbose > 0 {
			log.Println("cannot crop", fn)
		}
		return nil
	}
	thm_img := resize.Resize(thm_w, thm_h, thm_sub.SubImage(crop), resize.Lanczos2)
	thm_f, err := os.Create(thm_fn)
	defer thm_f.Close()
	if err != nil {
		return err
	}
	switch format {
	case "jpeg":
		jpeg.Encode(thm_f, thm_img, nil)
	case "png":
		png.Encode(thm_f, thm_img)
	default:
		return errors.New("unexpected format: " + format)
	}
	return nil
}

func main() {
	// command-line options
	f_verbose := flag.Bool("v", false, "Verbose error reporting")
	very_verbose := flag.Bool("V", false, "Very verbose error reporting")
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to file")
	f_thm_w := flag.Uint("tw", 256, "thumbnail width")
	f_thm_h := flag.Uint("th", 256, "thumbnail height")
	f_small_w := flag.Uint("sw", 800, "small width")
	f_small_h := flag.Uint("sh", 800, "small height")
	f_target := flag.String("o", "", "output directory for the gallery")
	title := flag.String("t", "", "title")
	flag.Parse()

	if *f_verbose {
		verbose = 1
	}
	if *very_verbose {
		verbose = 2
	}
	thm_w = *f_thm_w
	thm_h = *f_thm_h
	small_w = *f_small_w
	small_h = *f_small_h
	if *f_target == "" {
		log.Fatal("must specify an output directory using -o")
	}
	target = *f_target
	err := os.MkdirAll(target, 0755)
	if err != nil {
		log.Fatal("could not create output dir:", err)
	}
	index, err := os.Create(path.Join(target, "index.md"))
	if err != nil {
		log.Fatal("could not create output file:", err)
	}
	defer index.Close()

	var f *os.File
	// Profiler
	if *cpuprofile != "" {
		f, err = os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	pix = make([]Pix, 0, 100)

	// walk the current directory looking for image files
	err = filepath.Walk(".", func(fn string, i os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return thumbnail(fn, i)
	})
	if err != nil {
		log.Fatal("walk error:", err)
	}
	//log.Println(pix)
	if len(pix) == 0 {
		log.Fatalf("did not find any photos")
	}
	ib := bufio.NewWriter(index)
	fmt.Fprintln(ib, "+++")
	fmt.Fprintf(ib, "title = \"%s\"\n", *title)
	fmt.Fprintln(ib, time.Now().Format("date = \"2006-01-02\""))
	fmt.Fprintln(ib, "categories = [\"\"\"photos\"\"\"]")
	fmt.Fprintf(ib, "cover = \"%s\"\n", pix[0].Filename)
	fmt.Fprintln(ib, "+++\n")
	fmt.Fprintln(ib, "{{< wrap >}}")

	for _, p := range pix {
		// XXX should escape the filename and copyright
		fmt.Fprintf(ib, "{{< photo\n    href=\"%s\" largeDim=\"%dx%d\"\n    smallUrl=\"%s\" smallDim=\"%dx%d\"\n    thumbSize=\"%dx%d\" thumbUrl=\"%s\"\n    title=\"\"\n    caption=\"\"\n    alt=\"\"\n    copyright=\"%s\" >}}\n", p.Filename, p.Width, p.Height, p.Small, p.SmallWidth, p.SmallHeight, thm_w, thm_h, p.Thumbnail, p.Copyright)
	}

	fmt.Fprintln(ib, "{{< /wrap >}}")
	ib.Flush()

	return
}
