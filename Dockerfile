# Use the official image as a parent image.
FROM golang

# Set the working directory.
WORKDIR /src

# Copy the file from your host to your current location.
COPY . .

# Mount volume for DB
VOLUME ["~/botto-data/data", "/data"]

# Run the command inside your image filesystem.
RUN go build -o app
RUN chmod +x app

# Run the specified command within the container.
CMD [ "./app" ]