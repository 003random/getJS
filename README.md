# GetJS
[![License](https://img.shields.io/badge/license-MIT-_red.svg)](https://opensource.org/licenses/MIT)
[![contributions welcome](https://img.shields.io/badge/contributions-welcome-brightgreen.svg?style=flat)](https://github.com/003random/getJS/issues)

getJS is a tool to extract all the javascript files from a set of given urls.  

The urls can be piped to getJS, or you can specify a single url with the `-url` argument. getJS offers a range of options, varying from completing the urls, to resolving the urls to make sure they exist.

## Prerequisites

Make sure you have [GO](https://golang.org/) installed on your system.  

### Installing

getJS is written in GO. You can install it with `go get`:

```bash
go get github.com/003random/getJS
```

# Usage  
Note: When you supply urls from different sources, e.g. with stdin and an input file, it will add all the urls together :)  
Example: 

```bash
echo "https://github.com" | getJS -url=https://example.com -input=domains.txt
```

To get all  options, do:  
```bash
getJS -h
```
  

| Flag | Description | Example |
|------|-------------|---------|
| -url   | The url to get the javascript sources from | getJS -url=htt<span></span>ps://poc-server.com |
| -input   | Input file with urls            | getJS -input=domains.txt |
| -output   | The file where to save the output to        | getJS -output=output.txt |
| -plain  | Only output the results | getJS -plain |
| -silent  | Output nothing           | getJS -silent |
| -complete  | Complete the urls. e.g. /js/index.js -> htt<span></span>ps://example.<span></span>com/js/index.js  | getJS -complete |
| -resolve   | Resolve the output and filter out the non existing files (Can only be used in combination with -complete)   | getJS -complete -resolve |

## Examples  
  
 ![screenshot](https://poc-server.com/getJS/screenshot.png)

    
getJS supports stdin data. To pipe urls to getJS, use the following (-plain is optional).  

```bash
$ cat domains.txt | getJS -plain
```  
  
To save the js files, you can use:  
```bash
$ getJS -url=https://poc-server.com -plain | xargs wget
```
  
If you would like the output to be in JSON format, you can combine it with [@Tomnomnom's](https://github.com/tomnomnom) [toJSON](https://github.com/tomnomnom/hacks/tree/master/tojson):  
```bash
$ getJS -url=https://poc-server.com -plain | tojson
```  
  
To feed urls from a file use:  
```bash
$ getJS -input=domains.txt
```  
  
To save the results to a file, and don't display anything, use:  
```bash
$ getJS -url=https://poc-server.com -output=results.txt
```  
  
If you want to have a list of full urls as output use:  
```bash
$ getJS -url=domains.txt -complete
```  
  
If you want to only show the js files that resolve, use:  
```bash
$ getJS -url=domains.txt -complete -resolve
```  

## Built With

* [GO](http://golang.org/) - Golang
* [Goquery](https://github.com/PuerkitoBio/goquery) - HTML parser with syntaxes like jquery, in GO


## Contributing

You are free to submit any issues and/or pull requests :)

## License

This project is licensed under the MIT License.

## Acknowledgments

* [@jimen0](https://github.com/jimen0) for helping getting me started with GO
  
    
---
    
*This is my first tool written in GO. I created it to learn the language more. Please don't hate on my coding style xD (useful feeback is always welcome!)*
