## Use minimal base image
FROM alpine:3.19

# Set working directory
WORKDIR /app

# Copy prebuilt binaries
COPY drift-checker .
COPY test-binaries/ /app/test-binaries/

COPY .env /app/.env

# Set executable permission (if needed)
RUN chmod +x drift-checker
RUN chmod +x /app/test-binaries/*

# Default command
CMD ["./drift-checker"]






## Use minimal base image
#FROM alpine:3.19
#
## Set working directory
#WORKDIR /app
#
## Copy prebuilt binaries
#COPY drift-checker .
#COPY drift-checker.test .
#
#COPY .env /app/.env
#
## Copy the run-tests.sh script to the container
#COPY run-tests.sh run-tests.sh
#
## Set executable permissions
#RUN chmod +x run-tests.sh
#
## Set executable permission (if needed)
#RUN chmod +x drift-checker drift-checker.test
#
## Default command
#CMD ["./drift-checker"]
