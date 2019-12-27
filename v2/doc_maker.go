package main

import (
    "github.com/go-macaron/pongo2"
    "gopkg.in/russross/blackfriday.v2"
    "gopkg.in/macaron.v1"


    // "golang.org/x/net/html/atom"
    "io/ioutil"
    "io"
    "log"
    "os"
    "os/exec"
    "context"
    "time"
    "strings"
    "flag"
    "regexp"
    "bytes"
    "github.com/alecthomas/chroma"
    "github.com/alecthomas/chroma/styles"
    "github.com/alecthomas/chroma/lexers"
    "github.com/alecthomas/chroma/formatters/html"
)


func processHTML(in []byte, style *chroma.Style) (string) {
    buf2 := new(bytes.Buffer)
    buf2.WriteString("<style>")
    html.New(html.WithClasses(true)).WriteCSS(buf2, style)
    buf2.WriteString("</style>")
    out := buf2.String()

    out += "<section>"
    hasContent := false
    for _, line := range strings.Split(string(in), "\n") {
        if strings.HasPrefix(line, "<h") {
            if hasContent {
                if strings.HasPrefix(line, "<h1") || strings.HasPrefix(line, "<h2") {
                    out += "</section><div style='page-break-after: always;'></div><section>"
                } else {
                    out += "</section><section>"
                }
                log.Println("Switch section", line)
            }
            hasContent = false
        } else if strings.HasPrefix(line, "<p>============") {
            continue
        } else if !strings.HasPrefix(line, "<br>") && len(line) > 0 {
            hasContent = true
        }
        out += line
    }

    out += "</section>"

    generateToc(in)
    return string(out)
}

func highlight(in []byte) ([]byte, *chroma.Style) {
    style := styles.Get("monokai")
    if style == nil {
        style = styles.Fallback
    }
    formatter := html.New(html.WithClasses(true))

    re := regexp.MustCompile("```(?s)(.*?)```")
    re2 := regexp.MustCompile("```(?s)(\\S+)(.*?)```")

    out := re.ReplaceAllFunc(in, func(match []byte) []byte {
        res := re2.FindAllSubmatch(match, -1)
        lexer := lexers.Get(string(res[0][1]))
        iterator, _ := lexer.Tokenise(nil, strings.Trim(string(res[0][2]), "\r\n"))
        buf := new(bytes.Buffer)
        formatter.Format(buf, style, iterator)
        return []byte(strings.ReplaceAll(buf.String(), "\n","<br>"))
    })

    return out, style
}

func isolate(in, start, end string) string {
    begin := strings.Split(in, end)
    if len(begin) > 1 {
        end := strings.Split(begin[0], start)
        if len(end) > 1 {
            return begin[1]
        }
    }
    return ""
}

type tocNode struct {
    id string
    content string
    parent *tocNode
}

func generateToc(in []byte) string {
    out := "<ul class='toc'>"
    tocH1 := []*tocNode{}
    tocH2 := []*tocNode{}
    tocH3 := []*tocNode{}

    var cur *tocNode

    for _, line := range strings.Split(string(in), "\n") {
        if strings.HasPrefix(line, "<h1") {
            cur = &tocNode{
                id: isolate(line, "id=\"", "\""),
                content: isolate(line, ">", "<"),
                parent: nil,
            }
            tocH1 = append(tocH1, cur)
        } else if strings.HasPrefix(line, "<h2") {
            cur = &tocNode{
                id: isolate(line, "id=\"", "\""),
                content: isolate(line, ">", "<"),
                parent: cur,
            }
            tocH2 = append(tocH2, cur)
        } else if strings.HasPrefix(line, "<h3") {
            cur2 := &tocNode{
                id: isolate(line, "id=\"", "\""),
                content: isolate(line, ">", "<"),
                parent: cur,
            }
            tocH3 = append(tocH3, cur2)
        }
    }

    log.Println(tocH1, tocH2, tocH3)


    // for _, line := range strings.Split(string(in), "\n") {
    //     if strings.HasPrefix(line, "<h1") {
    //         content = isolate(line, ">", "<")
    //         id = isolate(line, "id=\"", "\"")
    //         out += "<li><a href=\"#" + id + "\">" + content + "</a></li>" // <span>" + page + "</span> page = "1"
    //     } else if strings.HasPrefix(line, "<h2") {
    //         content = isolate(line, ">", "<")
    //         id = isolate(line, "id=\"", "\"")
    //         out += "<li><a href=\"#" + id + "\">" + content + "</a></li>" // <span>" + page + "</span> page = "1"
    //     }
    // }
    return out + "</ul>"
}

// function generateToc(toc) {
//     res = []
//     toc.forEach(function(prim) {
//         res.push("<li><a href=\"#" + prim.id + "\">" + prim.text + "</a><span>" + prim.page + "</span></li>")
//         if (prim.subs.length > 0) {
//             res.push("<ul>");
//             prim.subs.forEach(function(sec) {
//                 res.push("<li><a href=\"#" + sec.id + "\">" + sec.text + "</a><span>" + sec.page + "</span></li>")
//             });
//             res.push("</ul>");
//         }
//     })
//     res.push("</ul>")
//     return res.join("");
// }


func main() {
    var source string
    fs := flag.NewFlagSet("", flag.ExitOnError)
	fs.StringVar(&source, "src", "", "Directory to serve")
    err := fs.Parse(os.Args[1:])
    if err != nil {
        log.Fatalln(err)
    }
    if _, err := os.Stat(source); os.IsNotExist(err) {
      log.Fatalln(err)
    }

    tmplPath, err := ioutil.TempDir("", "docmaker-cache*")
    if err != nil {
        log.Fatalln(err)
    }

    m := macaron.Classic()
    m.Use(macaron.Static("static", macaron.StaticOptions{Prefix: "static",}))
    m.Use(macaron.Static("pdf-cache", macaron.StaticOptions{Prefix: "output",}))
    m.Use(macaron.Static(source + "/img", macaron.StaticOptions{Prefix: "img",}))
    m.Use(pongo2.Pongoer())

    m.Get("/html/:file", func(ctx *macaron.Context) {
        data, err := ioutil.ReadFile(source + ctx.Params(":file") + ".md")
        if err != nil {
            ctx.Resp.WriteHeader(404)
            return
        }
        ctx.Data["Title"] = ctx.Params(":file") + ".md"

        data, style := highlight(data)

        ctx.Data["Body"] = processHTML(blackfriday.Run(data), style)
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
