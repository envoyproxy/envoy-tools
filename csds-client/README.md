# CSDS Client
[Client status discovery service (CSDS)](https://www.envoyproxy.io/docs/envoy/latest/api-v3/service/status/v3/csds.proto) is a generic xDS API that can be used to get information about data plane clients from the control plane’s point of view. It is useful to enhance debuggability of the service mesh, where lots of xDS clients are connected to the control plane.<br/>
The CSDS client is developed as a generic tool that can be used/extended to work with different xDS control planes.<br/>
For now, this initial version of this CSDS client only supports GCP's [Traffic Director](https://cloud.google.com/traffic-director).
<br/>Before you start, you'll need [Go](https://golang.org/) installed.

# Building
* Run `make` to install dependencies, build a binary under `GOPATH`, and run tests.<br>
  In this way, you'll need to export `GOPATH` as an environment variable when you install Go so that you can run the client with `csds-client <flag>` globally.
* Or, run `make init` to install dependencies and run `make build` to build a binary under the current path.<br>
  In this way, you can run the client with `./csds-client <flag>`.
* Run `make help` for other options.

# Running
* run with `csds-client <flag>`, e.g. <br/><br/>
   * auto authentication mode
   ```bash
   csds-client \
     -service_uri <uri> \
     -platform gcp \
     -authn_mode auto \
     -api_version v2 \
     -request_file <path to csds request yaml file>
  ```
   * jwt authentication mode
   ```bash
   csds-client \
     -service_uri <uri> \
     -platform gcp \
     -authn_mode jwt \
     -api_version v2 \
     -request_file <path to csds request yaml file> \
     -jwt_file <path to jwt key>
  ```

# Usage
Common options are exposed/controlled via command line flags, while control plane specific options are configured in a yaml file and are passed into [ClientStatusRequest](https://www.envoyproxy.io/docs/envoy/latest/api-v3/service/status/v3/csds.proto#service-status-v3-clientstatusrequest).
## Flags
* ***-service_uri***: the uri of the service to connect to 
   * If this flag is not specified, it will be set to *trafficdirector.googleapis.com:443* as default.
* ***-platform***: the platform (e.g. gcp, aws,  ...)
  * If this flag is not specified, it will be set to *gcp* as default.
  * This flag will be used for platform specific logic such as auto authentication.
* ***-authn_mode***: the method to use for authentication (e.g. auto, jwt, ...)
  * If this flag is not specified, it will be set to *auto* as default.
  * If it’s set to *auto*, the credentials will be obtained automatically based on different cloud platforms.
  * If it’s set to *jwt*, the credentials will be obtained from the jwt file which is specified by the ***-jwt_file*** flag.
* ***-api_version***: which xds api major version to use (e.g. v2, v3 ...)
  * If this flag is not specified, it will be set to *v2* as default.
* ***-jwt_file***: path of the jwt_file
* ***-request_file***: yaml file that defines the csds request
  * If this flag is missing, ***-request_yaml*** is required.
* ***-request_yaml***: yaml string that defines the csds request
  * If ***-request_file*** is also set, the values in this yaml string will override and merge with the request loaded from ***-request_file***. 
  * Because yaml is a superset of json, a json string may also be passed to ***-request_yaml***.
* ***-output_file***: file name to save configs returned by csds response
   * If this flag is not specified, the configuration will be output to stdout by default.
* ***-monitor_interval***: the interval of sending requests in monitor mode (e.g. 500ms, 2s, 1m, ...)
   * If this flag is not specified, the client will run only once.
   * If this flag is specified and the interval is greater than 0, the client will run continuously and send request based on the interval. Use `Ctrl+C` to exit.
* ***-visualization***: option to visualize the relationship between xDS resources
   * If this flag is not specified, the visualization mode is off by default
   * The client will generate a `.dot` file and save it as `config_graph.dot`, then it will open the browser window automatically to show the graph parsed by dot.
   * If the browser fails to open due to os version issue, you can copy the content in `config_graph.dot`, and then paste it in the edit box on the left of [Graphviz Online](https://dreampuf.github.io/GraphvizOnline/) or any other tools for [Graphviz](https://graphviz.org/) to show the graph of the dot file.
   * Each xDS node shown in the graph is labelled by index (e.g. LDS0, RDS0, RDS1,...) to make the graph more clear. The real name of xDS resource in config will show when the user hovers the mouse over each node.
   * If **the visualization mode** and **the monitor mode** are enabled together, the client will only save graph dot data for the latest response without opening the browser to avoid frequent pop-ups of the browser due to short monitor interval.
* ***-filter_mode***: the filter mode for the filter on Client ID to be returned (e.g. prefix, suffix, regex, ...)
   * If this flag is not specified, all Client ID will be returned.
* ***-filter_pattern***: the filter pattern for the filter on xDS nodes to be returned
   * This flag works with ***-filter_mode*** together.

## Output
```
Client ID                      xDS stream type                Config Status                           
<client_id>                    ADS                            LDS SYNCED
                                                              RDS SYNCED
                                                              CDS STALE
(Detailed Config:
 <detailed config>)
OR
(Config has been saved to <output_file>)
```