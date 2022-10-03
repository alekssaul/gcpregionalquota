# GCP Reqional Quota compare

Small CLI utility to compare quotas between two GCP regions within a project. 

## Build

Make sure you have a recent version of Golang installed 

```sh
git clone https://github.com/alekssaul/gcpregionalquota.git
cd gcpregionalquota
go build . 
```


## Run

```
NAME:
   GCP Regional Quota lookup -  

USAGE:
   gcpregionalquota --projectid project-123 --sourceregion us-east1 --targetregion us-central1

DESCRIPTION:
   Small CLI tool to compare GCP region quotas

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h            show help (default: false)
   --projectid value     GCP Project ID (required)
   --sourceregion value  GCP Source Region (required)
   --targetregion value  GCP Target Region (required)
```