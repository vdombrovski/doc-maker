package main

import (
    "github.com/go-macaron/pongo2"
    "gopkg.in/russross/blackfriday.v2"
    "gopkg.in/macaron.v1"
    "io/ioutil"
    "io"
    "log"
    "os"
    "os/exec"
    // "strings"
    "context"
    "time"
)

var source = "/home/korween/Projects/oio-back/oio-training/documentation/"
var tmplPath = "/home/korween/Projects/oio-back/doc-maker/v2/pdf-cache"

func main() {
    m := macaron.Classic()
    m.Use(macaron.Static("static", macaron.StaticOptions{Prefix: "static",}))
    m.Use(macaron.Static("pdf-cache", macaron.StaticOptions{Prefix: "output",}))
    m.Use(pongo2.Pongoer())

    m.Get("/html/:file", func(ctx *macaron.Context) {
        log.Println("registered access")
        data, err := ioutil.ReadFile(source + ctx.Params(":file") + ".md")
        if err != nil {
            ctx.Resp.WriteHeader(404)
            return
        }
        ctx.Data["Title"] = ctx.Params(":file") + ".md"
        ctx.Data["Body"] = string(blackfriday.Run(data))
        ctx.Data["Format"] = "fmt-html"
        ctx.HTML(200, "plate")
    })
    m.Get("/pdf/:file", func(ctx *macaron.Context) {
        tmpFile, err := ioutil.TempFile(tmplPath, "out*.pdf")
    	if err != nil {
            log.Println(err)
            ctx.Resp.WriteHeader(500)
            return
    	}
        tmpFileName := tmpFile.Name()

    	defer os.Remove(tmpFileName)

        err = renderPDF("http://127.0.0.1:4000/html/" + ctx.Params(":file"), tmpFileName)
        if err != nil {
            log.Println("ERROROOR", err)
            ctx.Resp.WriteHeader(500)
            return
        }

        f, err := os.Open(tmpFileName)
        if err != nil {
           log.Println(err)
           ctx.Resp.WriteHeader(500)
           return
        }
        defer f.Close()

        //Set header
        ctx.Resp.Header().Set("Content-type", "application/pdf")

        //Stream to response
        if _, err := io.Copy(ctx.Resp, f); err != nil {
           log.Println(err)
           ctx.Resp.WriteHeader(500)
        }
    })

    m.Run()
}

func renderPDF(input string, output string) (error) {
    // cmd := exec.Command(
    //     "/usr/bin/chrome",
    //     "--headless",
    //     "--run-all-compositor-stages-before-draw",
    //     "--no-margins",
    //     "--print-to-pdf=\"" + output + "\"",
    //     "\"" + input + "\"",
    // )

    ctx, _ := context.WithTimeout(context.Background(), time.Duration(5)*time.Second)
    log.Println(input)
    do := exec.CommandContext(ctx, "/usr/bin/chrome",
    "--headless",
    "--run-all-compositor-stages-before-draw",
    "--no-margins",
    "--print-to-pdf=" + output,
    input)

    stderr, err := do.StderrPipe()
	if err != nil {
        log.Println("err at pipe")
        return err
	}

	if err := do.Start(); err != nil {
        log.Println("err at start")
        return err
	}

	slurp, _ := ioutil.ReadAll(stderr)
    log.Println("SLURP", string(slurp))
	if err := do.Wait(); err != nil {
        log.Println("err at wait")
        return err
	}

    return nil

    // // out, err := sleep.Output()
    // log.Println(out)
    // if err != nil {
    //     return "", err
    // }
    // outFmt := strings.Trim(
    //     strings.TrimSuffix(string(out), "\n"), " ")
}
