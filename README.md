# GitBook to GitHub Wiki

[![build status](https://img.shields.io/travis/com/kataras/gitbook-to-wiki/master.svg?style=for-the-badge&logo=travis)](https://travis-ci.com/github/kataras/gitbook-to-wiki) [![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=for-the-badge)](https://goreportcard.com/report/github.com/kataras/gitbook-to-wiki) [![godocs](https://img.shields.io/badge/go-%20docs-488AC7.svg?style=for-the-badge)](https://pkg.go.dev/github.com/kataras/gitbook-to-wiki)

This CLI tool was initially created to generate the [Iris Wiki](https://github.com/kataras/iris/wiki). Works with the latest https://gitbook.com as of **2020**. 

GitBook is the best tool on writing & publishing markdown books. They have an excellent support team and they do offer premium plans for Free and Open Source Projects (I have seen it myself!).

## Installation

The only requirement is the [Go Programming Language](https://golang.org/dl).

```sh
$ go get github.com/kataras/gitbook-to-wiki
```

By navigating to the [Releases page](https://github.com/kataras/gitbook-to-wiki/releases) you can also **download the executable file** for your operating system. Note that, the $PATH system environment variable should contain an entry of the `gitbook-to-wiki` program you have just downloaded. Alternatively, copy-paste the `gitbook-to-wiki` executable to the current working directory.

## Getting Started

Navigate to the parent directory of your gitbook, e.g. `/home/me`.

Clone your repository's wiki:
```sh
$ git clone https://github.com/$username/$repo.wiki.git
```

The directory structure should look like that:
```
│
└───$repo.wiki
    |  .git
    |   Home.md
└───$gitbook
    |   SUMMARY.md
    ├───subdir
    |   ...other files
```

Open a terminal window and execute the `gitbook-to-wiki`:
```sh
$ gitbook-to-wiki -v --src=./$gitbook --dest=./$repo.wiki --remote=/$username/$repo/wiki
```

Push your changes to the `master` branch of your `$repo.wiki`:
```sh
$ git add .
$ git commit -S -m "add new sections"
$ git push -u origin master
```

Navigate to <https://github.com/$repo/wiki> and you should be able to read your GitBook as GitHub Wiki. Congrats, that's all!

## How it works?

1. Unescapes `\(should be unescaped\)` to `(should be unescaped)` (when you push GitBook's contents to a GitHub repository it automatically adds those escape characters)
2. Handles page references (`{% page-ref page="../dir/mypage.md" %}`)
3. Copies code snippets **untouchable** (that's trivial but important because older tools I've used reported and stopped the parsing because of a simple HTML code snippet!)
4. Handles **asset links**, both absolute(http...) and relative, e.g. `![](.gitbook/assets/image.png)`
5. Handles **section links**, e.g. `[page title](relative.md)` to `[[page title rel|relative]]`
6. Handles **sub directories and sub sections**, e.g. `responses/json.md` to `responses/responses-json.md` (so GitHub Wiki can see it as unique, as it does not support sub-directory-content).
7. Handles `SUMMARY.md` to `_Sidebar.md`, it is not just a simple copy-paste, a content like that:
```md
<!-- $gitbook/SUMMARY.md
Looks very nice on GitBook's navbar but
GitHub Wiki Sidebar wouldn't render it correctly.
-->
# Table of contents

* [What is Iris](README.md)

## 📌Getting started

* [Installation](getting-started/installation.md)
* [Quick start](getting-started/quick-start.md)
```

Is translated to:
```md
<!-- $repo.wiki/_Sidebar.md
GitHub Wiki Sidebar looks awesome now thanks to gitbook-to-wiki!
-->
* [[What is Iris|Home]]
* 📌Getting started
  * [[Installation|getting-started-installation]]
  * [[Quick start|getting-started-quick-start]]
```
8. And, of course, the `.git` directory is not copied or touched at all.

Don't hesitate to ask for more features. This tool works for the Git Book of a 18k starred project's documentation but if I missed something please [let me know](https://github.com/kataras/gitbook-to-wiki/issues/new).

## License

This software is created by [Gerasimos Maropoulos](https://twitter.com/MakisMaropoulos) and it is distributed under the [MIT License](LICENSE).
