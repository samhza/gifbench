package main

import (
	"bytes"
	"flag"
	"fmt"
	"image/png"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"text/tabwriter"
	"time"

	"go.samhza.com/esammy/memegen"
	"go.samhza.com/ffmpeg"
)

func init() {
	log.SetFlags(0)
}

type Program struct {
	Name    string
	Command CommandFunc
}

type CommandFunc func(in, out string, cap string) (*exec.Cmd, error)

func main() {
	v := flag.Bool("v", false, "verbose")
	tabbed := flag.Bool("tabbed", false,
		"print tabbed output instead of aligned output")
	flag.Parse()
	binprogs, err := os.ReadDir("bin")
	if err != nil {
		log.Fatalln(err)
	}
	files, err := os.ReadDir("in")
	if err != nil {
		log.Fatalln(err)
	}
	var progs []Program
	for _, f := range binprogs {
		progs = append(progs, Program{
			Name:    f.Name(),
			Command: binFunc(f),
		})
	}
	progs = append(progs, Program{
		Name:    "ffmpeg",
		Command: ffmpegFunc,
	})
	err = os.MkdirAll("out", 0777)
	if err != nil {
		log.Fatalln("creating output dir:", err)
	}
	var out io.Writer
	if *tabbed {
		out = os.Stdout
	} else {
		tw := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight)
		defer tw.Flush()
		out = tw
	}
	for _, f := range files {
		for _, p := range progs {
			if *v {
				log.Printf("testing %s with %s", p.Name, f.Name())
			}
			start := time.Now()
			outname := "out/" + p.Name + "_" + f.Name()
			cmd, err := p.Command("in/"+f.Name(), outname,
				"caption caption caption")
			if err != nil {
				log.Printf("error testing %s with %s: %s\n",
					p.Name, f.Name(), err)
			}
			fmt.Fprintf(out, "%s\t%s\t",
				f.Name(), p.Name)
			if err == nil {
				outf, err := os.Stat(outname)
				if err == nil {
					fmt.Fprintf(out, "%dns\t%d\t%d\t\n",
						time.Since(start).Nanoseconds(),
						cmd.ProcessState.SysUsage().(*syscall.Rusage).Maxrss,
						outf.Size())
				} else {
					fmt.Fprintf(out, "%dns\t%d\terror\t\n",
						time.Since(start).Nanoseconds(),
						cmd.ProcessState.SysUsage().(*syscall.Rusage).Maxrss)
				}
			} else {
				fmt.Fprintf(out, "error\t\t\t\n")
			}
		}
	}
}

func binFunc(f fs.DirEntry) CommandFunc {
	return func(inpath, outpath, cap string) (*exec.Cmd, error) {
		cmd := exec.Command("bin/"+f.Name(), cap)
		in, err := os.Open(inpath)
		if err != nil {
			return nil, err
		}
		defer in.Close()
		cmd.Stdin = in
		out, err := os.Create(outpath)
		if err != nil {
			return nil, err
		}
		defer out.Close()
		cmd.Stdout = out
		return cmd, cmd.Run()
	}
}

func ffmpegFunc(in string, out string, cap string) (*exec.Cmd, error) {
	input := ffmpeg.Input{Name: in}
	var v ffmpeg.Stream
	size, err := probeSize(in)
	if err != nil {
		return nil, err
	}
	img, pt := memegen.Caption(size.Width, size.Height, cap)
	pR, pW, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	defer pW.Close()
	enc := png.Encoder{CompressionLevel: png.NoCompression}
	go func() {
		enc.Encode(pW, img)
		pW.Close()
	}()
	imginput := ffmpeg.InputFile{File: pR}
	v = ffmpeg.Overlay(imginput, input, -pt.X, -pt.Y)
	one, two := ffmpeg.Split(v)
	palette := ffmpeg.PaletteGen(two)
	v = ffmpeg.PaletteUse(one, palette)
	fcmd := &ffmpeg.Cmd{}
	streams := []ffmpeg.Stream{v}
	outopts := []string{"-f", "gif", "-shortest"}
	fcmd.AddOutput(out, outopts, streams...)
	cmd := fcmd.Cmd()
	cmd.Args = append(cmd.Args, "-y", "-loglevel", "error", "-shortest")
	cmd.Args = append(cmd.Args, "-gifflags", "-offsetting")
	return cmd, cmd.Run()
}

type Size struct {
	Width  int
	Height int
}

func probeSize(path string) (*Size, error) {
	cmd := exec.Command(
		"ffprobe",
		"-v", "quiet",
		"-read_intervals", "%+#1", // 1 frame only
		"-select_streams", "v:0",
		"-print_format", "default=noprint_wrappers=1",
		"-show_entries", "stream=width,height", path,
	)

	// The output is small enough, so whatever.
	b, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute FFprobe: %w", err)
	}

	var size Size

	for _, t := range bytes.Fields(b) {
		p := bytes.Split(t, []byte("="))
		if len(p) != 2 {
			return nil, fmt.Errorf("invalid line: %q", t)
		}

		i, err := strconv.Atoi(string(p[1]))
		if err != nil {
			return nil, fmt.Errorf("failed to parse int from line %q: %w", t, err)
		}

		switch string(p[0]) {
		case "width":
			size.Width = i
		case "height":
			size.Height = i
		}
	}

	return &size, nil
}
