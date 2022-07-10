# GOWERC

clone of http://werc.cat-v.org/ in go.

## features
* virtual hosting
* render markdown, html, plain text
* serve files
* read content from zip file

## building

### with go

	go install git.froth.zone/sam/go2werc

### in docker

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

	docker run --rm --name gowerc -v $PWD:/opt mischief/gowerc -root /opt/werc.zip

