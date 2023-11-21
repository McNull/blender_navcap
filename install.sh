#!/bin/bash
go build -o blender_navcap

# Copy the service file to the systemd directory
sudo rm -f /etc/systemd/system/blender_navcap.service
sudo cp blender_navcap.service /etc/systemd/system/

# Copy the blender_navcap executable to /usr/bin
sudo rm -f /usr/bin/blender_navcap
sudo cp ./blender_navcap /usr/bin/

# Set the executable permission
sudo chmod +x /usr/bin/blender_navcap

# Reload the systemd daemon to recognize the new service
sudo systemctl daemon-reload

# Enable the service to start on boot
sudo systemctl enable blender_navcap

# Start the service
sudo systemctl start blender_navcap