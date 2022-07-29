# Building Mailpit from source

Go (>= version 1.8) and npm are required to compile mailpit from source.

```
git clone git@github.com:axllent/mailpit.git
cd mailpit
```

## Building the UI

The Mailpit web user interface is built with node. In the project's root (top) directory run the following to install the required node modules:


### Installing the node modules
```
npm install
```


### Building the web UI

```
npm run build
```

You can also run `npm run watch` which will watch for changes and rebuild the HTML/CSS/JS automatically when changes are detected.
Please note that you must restart Mailpit (`go run .`) to run with the changes.


## Build the mailpit binary

One you have the assets compiled, you can build mailpit as follows:
```
go build -ldflags "-s -w"
```

## Building a stand-alone sendmail binary

This step is unnecessary, however if you do not intend to either symlink `sendmail` to mailpit or configure your existing sendmail to route mail to mailpit, you can optionally build a stand-alone sendmail binary.

```
cd sendmail
go build -ldflags "-s -w"
```
