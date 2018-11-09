# GetJS
[![License](https://img.shields.io/badge/license-MIT-_red.svg)](https://opensource.org/licenses/MIT)
[![contributions welcome](https://img.shields.io/badge/contributions-welcome-brightgreen.svg?style=flat)](https://github.com/003random/getJS/issues)

getJS is a tool to extract all the javascript files from a set of given urls.  

The urls can also be piped to getJS, or you can specify a singel url with the -url argument. getJS offers a range of options, ranging from completing the urls, to resolving the files.

## Prerequisites

Make sure you have [GO](https://golang.org/) installed on your system.  

### Installing

getJS is written in GO. You can install it with `go get`:

```
go get github.com/003random/getJS
```

# Usage

```bash
getJS -h
```
This will display help for the tool. Here are all the switches it supports.

| Flag | Description | Example |
|------|-------------|---------|
| -url   | The url to get the javascript sources from | getJS -url=https://poc-server.com |
| -input   | Input file with urls            | getJS -input=domains.txt |
| -output   | The file where to save the output to        | getJS -output=output.txt |
| -plain  | Only output the results | getJS -plain |
| -silent  | Output nothing           | getJS -silent |
| -complete  | Complete the urls. e.g. /js/index.js -> https://example.com/js/index.js  | getJS -complete |
| -resolve   | Resolve the output and filter out the non existing files (Can only be used in combination with -complete)   | getJS -complete -resolve |


## Built With

* [GO](http://golang.org/) - GOlanguage
* [Goquery](https://github.com/PuerkitoBio/goquery) - HTML parser with syntaxes like jquery, in GO


## Contributing

You are free to submit any issues of pull requests :)

## License

This project is licensed under the MIT License.

## Acknowledgments

* [@jimen0](https://github.com/jimen0) for helping getting me started with GO

