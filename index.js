#!/usr/bin/env node
const pdf = require('html-pdf');
const fs = require('fs');
const http = require('http');
const path = require('path');
const url = require('url');
const process = require('process');
const marked = require('marked');
const highlight = require('highlight.js');


function loadConfig() {
    var conf = fs.readFileSync("./config.json");
    return JSON.parse(conf);
}

marked.setOptions({
    renderer: new marked.Renderer(),
    highlight: function(code, lang) {
        if(lang != "") {
            if(lang == "sh")
                lang = "bash";
            return highlight.highlight(lang, code).value;
        }
        return highlight.highlightAuto(code).value;
    },
    pedantic: false,
    gfm: true,
    breaks: true,
    sanitize: false,
    smartLists: true,
    smartypants: false,
    xhtml: false
});

if (process.argv.length != 3) {
    console.log("Usage: " + process.argv[1] + " [path to source files]")
    process.exit()
}

const SRC_DIR = path.resolve(process.argv[2]);
const CONFIG = loadConfig();
const ASSET_DIR = path.resolve(CONFIG.path.assets);
const IMG_DIR = path.resolve(CONFIG.path.images);
const PDF_CONFIG = {
    base: "file://" + ASSET_DIR + "/",
    border: CONFIG.pdf.border,
    paginationOffset: 1,
    header: {
        height: CONFIG.pdf.header.height,
        contents: '<div style="' + CONFIG.pdf.header.slide + '">' + CONFIG.pdf.header.text + '</div>'
    },
    footer: {
        height: CONFIG.pdf.footer.height,
        contents: {
            default: '<div style="' + CONFIG.pdf.footer.style + '">' + CONFIG.pdf.footer.text + '{{page}}/{{pages}}</div>'
        }
    }
}
const PDF_SLIDE_CONFIG = {
    base: "file://" + ASSET_DIR + "/",
    border: CONFIG.slide.border,
    orientation: "landscape",
    paginationOffset: 1,
    header: {
        height: CONFIG.slide.header.height,
        contents: '<div style="' + CONFIG.slide.header.style + '">' + CONFIG.slide.header.text + '</div>'
    },
    footer: {
        height: CONFIG.slide.footer.height,
        contents: {
            default: '<div style="' + CONFIG.slide.footer.style + '">' + CONFIG.slide.footer.text + '{{page}}/{{pages}}</div>'
        }
    }
}



if(!fs.existsSync(SRC_DIR)) {
    console.log("no such path", process.argv[2])
    process.exit()
}

console.log("Serving", SRC_DIR)
console.log("Listening on", CONFIG.port)

function readFile(fp) {
    if(!fs.existsSync(fp))
        return null;
    return fs.readFileSync(fp, 'utf8');
}

function isHeader(line, primary) {
    if(!line) return;
    if(primary == "both") {
        return line.startsWith("<h1") || line.startsWith("<h2") || line.startsWith("<h3");
    }
    else if(primary)
        return line.startsWith("<h1") || line.startsWith("<h2");
    return line.startsWith("<h3");
}

function html(title, body, slideMode) {
    var plate = readFile(ASSET_DIR + "/plate.html");
    if(!plate) return null;

    var lines = body.split('\n');
    var buf = []
    var lastHeaderPrim = "";
    var lastHeaderSec = "";
    var needsHeaderPrim = false;
    var needsHeaderSec = false;
    var needsBreak = false;
    var l = lines.length;
    for (i in lines) {
        needsBreak = false;
        var line = lines[i];
        if ((!slideMode) && isHeader(lines[parseInt(i)+1], "both")) {
            needsBreak = true;
        }
        if (isHeader(line, true)) {
            lastHeaderPrim = line;
            needsHeaderPrim = false;
            needsHeaderSec = false;
        }
        else {
            if (isHeader(line, false)) {
                lastHeaderSec = line;
                needsHeaderSec = false;
            }
            if(needsHeaderPrim) {
                if(slideMode)
                    buf.push(lastHeaderPrim);
                needsHeaderPrim = false;
            }
            if(needsHeaderSec) {
                if(slideMode)
                    buf.push(lastHeaderSec);
                needsHeaderSec = false;
            }
            if (line.match(/={10,}/g)) {
                var rpl = "<div class='" + ((needsBreak)?"slidebreak":"") + "' style='page-break-after: always;'></div>";
                line = line.replace(/={10,}/g, rpl);
                needsHeaderPrim = true;
                needsHeaderSec = true;
            }
        }
        buf.push(line)
    }
    buf = buf.join("\n")
    return plate.replace("{{title}}", title).replace("{{body}}", buf);
}

http.createServer(function(req, res) {
    var file = path.parse(url.parse(req.url).path);
    if (req.url == "/") {
        var body = "<ul>";
        var files = fs.readdirSync(SRC_DIR).forEach(function(f) {
            var fp = path.parse(f);
            if(fp.ext != ".md") return;
            body += `<li>item&nbsp;<a href='item.md'>Markdown</a>
                | <a href='item.html'>HTML</a> | <a href='item.pdf'>PDF</a>
                | <a href='item.slide'>Slides</a></li>`.replace(/item/g, fp.name);
        });
        body += "</ul>";
        res.writeHead(200, {'Content-Type': 'text/html'});
        return res.end(html("index", body));
    }
    switch(file.ext) {
        case ".png": case ".jpg": case ".svg":
            var imagePath = path.join(IMG_DIR, file.base);
            if(!fs.existsSync(imagePath)) break;
            var fileStream = fs.createReadStream(imagePath);
            res.writeHead(200, {"Content-Type": "image/" + file.ext.substring(1)});
            return fileStream.pipe(res);
        case ".css":
            var body = readFile(path.join(ASSET_DIR, file.base));
            if (!body) break;
            res.writeHead(200, {'Content-Type': 'text/css'});
            return res.end(body);
        case ".js":
            var body = readFile(path.join(ASSET_DIR, file.base));
            if (!body) break;
            res.writeHead(200, {'Content-Type': 'text/javascript'});
            return res.end(body);
        case ".md":
            var body = readFile(path.join(SRC_DIR, file.base));
            if (!body) break;
            res.writeHead(200, {'Content-Type': 'text/plain'});
            return res.end(body);
        case ".html":
            var body = readFile(path.join(SRC_DIR, file.name + ".md"));
            if (!body) break;
            res.writeHead(200, {'Content-Type': 'text/html'});
            return res.end(html(file.name, marked(body)));
        case ".pdf": case ".slide":
            var body = readFile(path.join(SRC_DIR, file.name + ".md"));
            if(!body) break;
            var bodyHTML = html(file.name, marked(body), (file.ext == ".slide"));
            if(!bodyHTML) break;
            var config = (file.ext == ".slide")?PDF_SLIDE_CONFIG:PDF_CONFIG;
            res.writeHead(200, {'Content-Type': 'application/pdf'});
            pdf.create(bodyHTML, config).toStream(function(err, stream) {
                if (err) return res.writeHead(500).end(err);
                stream.pipe(res);
            });
            return;
    }
    res.writeHead(404).end();

}).listen(CONFIG.port);
