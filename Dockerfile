FROM scratch
MAINTAINER Nick Owens <mischief@offblast.org>

ADD bin/gowerc /gowerc

ENTRYPOINT ["/gowerc"]

