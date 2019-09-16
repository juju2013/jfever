# Jfever

Jfever is (yet another) simple *live* static web site generator

Original idea and implementation from [PuerkitoBio](https://github.com/PuerkitoBio/trofaf).

Install using: `go get git.inexacte.science/juju/jfever`

## Features

* Static site generator, the generated content can be copied and served by any web server
* Plain text file and directory, no configuration file, no database. Use your favorite text editor/file manager to organize your site
* [Markdown][1] syntax, [Amber][2] template
* Integrated web server to see your changes *live*
* Super easy deployment, no dependency hell, just one static binary to copy


## Description

You'll need the following directories (their name can be changed via command line options):

* `out/`: Generated content
* `src/`: Source files for web pages, directory tree produce site navigation path
* `template/`: Your amber templates
* `static/`: Your static contents

Any change in the last 3 directories (or their sub-directories) will trigger the rebuild process and regenerate all contents. 
Direcotry creation, file creation and deletion will be reflected in out/ directory.

Jfever only cares about `*.md` files in the src directory, and about `*.amber` ([Amber templates][2]) in templates directory, 
any other files will be copied as is to out/ directory. Hidden files starting with `.` are ignored.

## Command-line Options

The following options can be set at the command-line: 

```
  -p, --port=          the port to use for the web server (default: 9000)
  -g, --generate-only  generate the static site and exit
  -G, --no-generation  when set, the site is not automatically generated
  -n, --site-name=     the name of the site (default: Site Name)
  -t, --tag-line=      the site's tag line
  -r, --recent-posts=  the number of recent posts to send to the templates (default: 5)
  -b, --base-url=      the base URL of the web site (default: http://localhost)
  -d, --debug          Enable debug output
  -s, --src=           the source sub-dir name (default: src)
  -o, --out=           the output sub-dir name (default: out)
  -a, --template=      the template sub-dir name (default: templates)
  -i, --static=        static content to be copied to Out/ (default: static)
```

## Front matter

Jfever uses *YAML front matter* to get metadata for a post. This is a complicated way to say that you have to add blocks of text like this at the start of your posts:

```
---
Title: My title
Description: My short-ish description of the post.
Author: Me
Date: 2013-07-14
Lang: en
---

# Here is my post!

Etc.
```

The three dashes delimit the front matter. It must be there, beginning and end. Between the dashes, the part before the colon `:` is the key, and after is the value. Simple as that.

Keys `Title`, `Description`, `Author` and `Date` are mandatory. 

Valid date formats are `2006-01-02`, `2006-01-02 15h` (or `2006-01-02 8h`), `2006-01-02 15:04` (or `2006-01-02 8:17`) or the RCF3339 format (`2013-08-06T17:48:01-05:00`).

Key `Template` can be used to choose the template (without the .amber extension) to use, default to `default`.

Key `Index` will save also the page as `index.html`, its value will be ignored.

You can define any `xxx` key in the front matter and use that key/value in your template.

## Demo

There a simple demonstration site under the `examples/amber` directory.

## License

The [BSD 3-Clause License][4].

[1]: http://daringfireball.net/projects/markdown/syntax
[2]: https://github.com/eknkc/amber
[3]: http://golang.org/pkg/html/template/
[4]: http://opensource.org/licenses/BSD-3-Clause
