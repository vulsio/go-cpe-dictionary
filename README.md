# go-cpe-dictionary

This is tool to build a local copy of the CPE (Common Platform Enumeration) [1].

> CPE is a structured naming scheme for information technology systems, software, and packages. Based upon the generic syntax for Uniform Resource Identifiers (URI), CPE includes a formal name format, a method for checking names against a system, and a description format for binding text and tests to a name.

go-cpe-dictionary download CPE data from NVD (National Vulnerabilities Database) [2].
Copy is generated in sqlite format.

[1] https://nvd.nist.gov/cpe.cfm  
[2] https://en.wikipedia.org/wiki/National_Vulnerability_Database  

[![asciicast](https://asciinema.org/a/asvc87lbpad5999shqk0xvtc0.png)](https://asciinema.org/a/asvc87lbpad5999shqk0xvtc0)

## Install requirements

go-cpe-dictionary requires the following packages.

- sqlite
- git
- gcc
- go v1.7 or later
    - https://golang.org/doc/install

```bash
$ sudo yum -y install sqlite git gcc
$ wget https://storage.googleapis.com/golang/go1.14.2.linux-amd64.tar.gz
$ sudo tar -C /usr/local -xzf go1.14.2.linux-amd64.tar.gz
$ mkdir $HOME/go
```
Put these lines into /etc/profile.d/goenv.sh

```bash
export GOROOT=/usr/local/go
export GOPATH=$HOME/go
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
```

Set the OS environment variable to current shell
```bash
$ source /etc/profile.d/goenv.sh
```

## Deploy go-cpe-dictionary

To install:

```bash
$ mkdir -p $GOPATH/src/github.com/kotakanbe
$ cd $GOPATH/src/github.com/kotakanbe
$ git clone https://github.com/kotakanbe/go-cpe-dictionary.git
$ cd go-cpe-dictionary
$ make install
```

Fetch CPE data from NVD. It takes about 1 minutes.  

```bash
$ go-cpe-dictionary fetch nvd
... snip ...
$ ls -alh cpe.sqlite3
-rw-r--r-- 1 ec2-user ec2-user 7.0M Mar 24 13:20 cpe.sqlite3
```

Now we have a local copy of CPE data in sqlite3.  

# How to search CPE name by application name

This example use [Peco](https://github.com/peco/peco) for incremental search.

```
$ ls cpe.sqlite3
cpe.sqlite3
$ sqlite3 ./cpe.sqlite3 'select cpe_uri from categorized_cpes' | peco
```

[![asciicast](https://asciinema.org/a/asvc87lbpad5999shqk0xvtc0.png)](https://asciinema.org/a/asvc87lbpad5999shqk0xvtc0)


# Usage:

```console
$ go-cpe-dictionary fetch --help
Fetch the data of CPE

Usage:
  go-cpe-dictionary fetch [command]

Available Commands:
  jvn         Fetch CPE from JVN
  nvd         Fetch CPE from NVD

Flags:
  -h, --help          help for fetch
      --stdout        display all CPEs to stdout
      --threads int   The number of threads to be used (default 4)
      --wait int      Interval between fetch (seconds)

Global Flags:
      --config string       config file (default is $HOME/.go-cpe-dictionary.yaml)
      --dbpath string       /path/to/sqlite3 or SQL connection string (default "$PWD/cpe.sqlite3")
      --dbtype string       Database type to store data in (sqlite3, mysql, postgres or redis supported) (default "sqlite3")
      --debug               debug mode (default: false)
      --debug-sql           SQL debug mode
      --http-proxy string   http://proxy-url:port (default: empty)
      --log-dir string      /path/to/log (default "/var/log/go-cpe-dictionary")
      --log-json            output log as JSON

Use "go-cpe-dictionary fetch [command] --help" for more information about a command.

$ go-cpe-dictionary server --help
Start CPE dictionary HTTP server

Usage:
  go-cpe-dictionary server [flags]

Flags:
      --bind string   HTTP server bind to IP address (default: loop back interface (default "127.0.0.1")
  -h, --help          help for server
      --port string   HTTP server port number (default: 1328 (default "1328")

Global Flags:
      --config string       config file (default is $HOME/.go-cpe-dictionary.yaml)
      --dbpath string       /path/to/sqlite3 or SQL connection string (default "$PWD/cpe.sqlite3")
      --dbtype string       Database type to store data in (sqlite3, mysql, postgres or redis supported) (default "sqlite3")
      --debug               debug mode (default: false)
      --debug-sql           SQL debug mode
      --http-proxy string   http://proxy-url:port (default: empty)
      --log-dir string      /path/to/log (default "/var/log/go-cpe-dictionary")
      --log-json            output log as JSON
```

----

# Misc

- HTTP Proxy Support  
If your system is behind HTTP proxy, you have to specify --http-proxy option.

- How to cross compile
    ```bash
    $ cd /path/to/your/local-git-reporsitory/go-cpe-dictionary
    $ GOOS=linux GOARCH=amd64 go build -o cvedict.amd64
    ```

- Debug  
Run with --debug, --debug-sql option.

----

# Data Source

- [NVD](https://nvd.nist.gov/)
- [JVN](https://jvndb.jvn.jp/)

----

# Authors

kotakanbe ([@kotakanbe](https://twitter.com/kotakanbe)) created go-cpe-dictionary and [these fine people](https://github.com/kotakanbe/go-cpe-dictionary/graphs/contributors) have contributed.

----

# Contribute

1. fork a repository: github.com/kotakanbe/go-cpe-dictionary to github.com/you/repo
2. get original code: go get github.com/kotakanbe/go-cpe-dictionary
3. work on original code
4. add remote to your repo: git remote add myfork https://github.com/you/repo.git
5. push your changes: git push myfork
6. create a new Pull Request

- see [GitHub and Go: forking, pull requests, and go-getting](http://blog.campoy.cat/2014/03/github-and-go-forking-pull-requests-and.html)

----

# Licence

Please see [LICENSE](https://github.com/kotakanbe/go-cpe-dictionary/blob/master/LICENSE).

----

# Additional License

- [NVD](https://nvd.nist.gov/faq)
>How can my organization use the NVD data within our own products and services?  
> All NVD data is freely available from our XML Data Feeds. There are no fees, licensing restrictions, or even a requirement to register. All NIST publications are available in the public domain according to Title 17 of the United States Code. Acknowledgment of the NVD  when using our information is appreciated. In addition, please email nvd@nist.gov to let us know how the information is being used.  
 
