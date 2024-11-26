# Bastion Server

## Getting started

- go mod tidy
- go run main.go


## Routes

### Devices

- Get list of devices: Websocket /devices/ws
- Create a device: POST /add-device
- Update a device: PUT /edit-device/:id
- Delete a device: DELETE /delete-device/:id


### Change Log

- Get list of Devices: Websocket /change-log/ws
