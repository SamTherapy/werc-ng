# GOWERC

clone of http://werc.cat-v.org/ in go.

## features
* virtual hosting
* render markdown, html, plain text
* serve files
* read content from zip file

## building

### with go install

	go install git.froth.zone/sam/go2werc@latest

<!-- ### in docker

check out the repo and run:

	make

## configuration

see the example configuration files in [werc](werc)

### using a zip file

zip a directory up and pass it to gowerc's `-root` option.

	zip -r werc.zip werc

## running

	gowerc -l :80 -root werc/

with a zip file:

	gowerc -l :80 -root werc.zip

### in docker

	docker run --rm --name gowerc -v $PWD:/opt mischief/gowerc -root /opt/werc.zip -->

## License
Werc was originally put into the public domain. Since the public domain isn't a thing
outside of the US, 0BSD is used instead. It's basically the public domain but the EU 
actually recognizes it.