# Using official python runtime base image
FROM python:2.7-alpine

# Set the application directory
WORKDIR /src

# Make port 80 available for links and/or publish
EXPOSE 80

# Copy our code from the current folder to /app inside the container
ADD . /src

RUN pip install -r requirements.txt

# Define our command to be run when launching the container
CMD ["python", "app.py"]
