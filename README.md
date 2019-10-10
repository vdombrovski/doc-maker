Doc-maker: a tool to easily create and serve styled pdf and html files
===

Description
---

This tool will serve your markdown documents in different formats. These include: html,pdf-portrait,pdf-landscape

Setup
---
```sh
yum install nodejs bzip2 -y
npm i
cp config.sample.json config.json
node ./index.js [path to markdown source files]
```

Then head to localhost:8080 to get an index of all your files with links to different formats.

Editing
---

Markdown files are formatted using classic GitHub like markdown. The only addition is the special pagebreak macro,
which you can invoke using '==========================' (at least 10 equal signs).

In slide mode, this will format your slides accordingly

Styles are available in the assets file, and are formatted using classic CSS, which you can modify. Same styles apply
for all formats.

You can use remote or local images (.jpg/.png/.svg) by linking them as follows:

```sh
![myImage](../img/test.png)
<img src="../img/test.png"/ width="200px">
```

Images must be placed inside the ./img directory (by default)

Other options are available to be configured via the config.json file.
