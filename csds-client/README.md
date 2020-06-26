# CSDS Client
[Client status discovery service (CSDS)](https://www.envoyproxy.io/docs/envoy/latest/api-v2/service/status/v2/csds.proto) is a generic xDS API that can be used to get information about data plane clients from the control plane’s point of view. It is useful to enhance debuggability of the service mesh, where lots of xDS clients are connected to the control plane.<br/>
The CSDS client is developed as a generic tool that can be used/extended to work with different xDS control planes.<br/>
For now, this initial version of this CSDS client only support GCP's [Traffic Director](https://cloud.google.com/traffic-director).
<br/>Before you start, you'll need [Go](https://golang.org/) installed.

# Building
* install dependencies using `go get`.

# Running
* run with `go run main.go <flag>`, e.g. <br/><br/>
   * running with auto authentication mode, run with 
   ```
   go run main.go -service_uri trafficdirector.googleapis.com:443 -cloud_platform gcp -authn_mode auto -api_version v2 -csds_request_yaml <path to csds request yaml file>
  ```
   * running with jwt authentication mode, run with 
   ```
   go run main.go -service_uri trafficdirector.googleapis.com:443 -cloud_platform gcp -authn_mode jwt -api_version v2 -csds_request_yaml <path to csds request yaml file> -jwt_file <path to jwt key>
  ```

# Usage
Options that are common can be exposed/controlled through command line flags, and options that are specific to control planes can be configured in a yaml file that can be parsed into ClientStatuRequest.  
## Flags
* ***-service_uri***: the uri of the service to connect to 
   * If this flag is not specified, it will be set to *trafficdirector.googleapis.com:443* as default.
* ***-cloud_platform***: the cloud platform (e.g. gcp, aws,  ...)
  * If this flag is not specified, it will be set to *gcp* as default.
  * This flag will be used for platform specific logic such as auto authentication.
* ***-authn_mode***: the method to use for authentication (e.g. auto, jwt, ...)
  * If this flag is not specified, it will be set to *auto* as default.
  * If it’s set to *auto*, the credentials will be obtained automatically based on different cloud platforms.
  * If it’s set to *jwt*, the credentials will be obtained from the jwt file which is specified by the ***-jwt_file*** flag.
* ***-api_version***: which xds api major version to use (e.g. v2, v3 ...)
  * If this flag is not specified, it will be set to *v2* as default.
* ***-jwt_file***: path of the jwt_file
* ***-csds_request_yaml***: yaml file that defines the csds request
* ***-file_to_save_config***: file name to save configuration
   * If this flag is not specified, the file name will be generated automatically and the config will be saved to `<client_id>_<xds_stream_type>_config.json`.

## Output
```
Client ID                                          xDS stream type    Config                            
<client_id>                                        ADS                myconfig.json
```