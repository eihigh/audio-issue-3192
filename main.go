package main

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
)

var (
	file     = flag.String("file", "cat.ogg", "path to play")
	typ      = flag.String("type", "Reader", "type of audio source (Reader or ReadSeeker)")
	panics   = flag.Bool("panics", false, "panics on eof")
	context  = flag.Int("context", 48000, "sampleRate of audio context")
	resample = flag.Int("resample", 48000, "sampleRate of resampling")
	listen   = flag.Int("listen", 5, "listen time in seconds")
)

//go:embed *.ogg *.mp3
var fsys embed.FS

type reader struct {
	src io.ReadSeeker
}

func (r *reader) Read(p []byte) (n int, err error) {
	n, err = r.src.Read(p)
	if *panics && err == io.EOF {
		panic("eof")
	}
	return n, err
}

type readSeeker struct {
	src io.ReadSeeker
}

func (r *readSeeker) Read(p []byte) (n int, err error) {
	n, err = r.src.Read(p)
	if *panics && err == io.EOF {
		panic("eof")
	}
	return n, err
}

func (r *readSeeker) Seek(offset int64, whence int) (int64, error) {
	return r.src.Seek(offset, whence)
}

func run() error {
	flag.Parse()

	b, err := fsys.ReadFile(*file)
	if err != nil {
		return err
	}

	var r io.Reader
	switch *typ {
	case "Reader":
		r = &reader{src: bytes.NewReader(b)}
	case "ReadSeeker":
		r = &readSeeker{src: bytes.NewReader(b)}
	default:
		panic("invalid -type")
	}

	ctx := audio.NewContext(*context)

	f := func(r io.ReadSeeker) (*audio.Player, error) {
		p, err := ctx.NewPlayer(r)
		if err != nil {
			return nil, err
		}
		b, err := io.ReadAll(r)
		if err != nil {
			return nil, err
		}
		// if err := os.WriteFile("out.pcm", b, 0644); err != nil {
		// 	return nil, err
		// }
		l := 20
		if len(b) < l {
			l = len(b)
		}
		fmt.Println(b[:l])
		return p, nil
	}

	var p *audio.Player
	switch {
	case strings.HasSuffix(*file, ".ogg"):
		stream, err := vorbis.DecodeWithSampleRate(*resample, r)
		if err != nil {
			return err
		}
		fmt.Println(stream.Length())
		p, err = f(stream)
		if err != nil {
			return err
		}

	case strings.HasSuffix(*file, ".mp3"):
		stream, err := mp3.DecodeWithSampleRate(*resample, r)
		if err != nil {
			return err
		}
		fmt.Println(stream.Length())
		p, err = f(stream)
		if err != nil {
			return err
		}

	default:
		panic("invalid file format")
	}

	p.SetVolume(0.3)
	p.Play()

	<-time.After(time.Duration(*listen) * time.Second)
	return nil
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}
