# GoHead on Render, built by render.yaml at the repository root.
#
# The released image, run as root: Render mounts persistent disks owned by
# root, and the image's non-root user (65532) could not write the database,
# the uploads or the generated JWT secret to it. The image has no shell or
# package manager, so root gives little to work with.
#
# 0.6 follows the 0.6.x releases: on Render, "Manual Deploy → Clear build
# cache & deploy" picks up the latest one.
ARG GOHEAD_VERSION=0.6
FROM ghcr.io/gohead-cms/gohead:${GOHEAD_VERSION}
USER 0:0
CMD ["standalone"]
