let EventEmitter = require('events')
let dateFormat = require('dateformat')

const http = require('http');

class Clusters extends EventEmitter {
  constructor(options) {
    super()
    this.log = options.log
    this.options = options || {}
    this.clustersURI = this.options.clustersURI || 'http://localhost:9999/clusters'
    this.bufferSize = this.options.bufferSize || 20
    this.pollingInterval = this.options.pollingInterval || 1000
    this.bufferIdx = -1
    this.clusters = {}
    this.times = []
  }

  /**
   * return an array of cluster names from the last call to the clusters endpoint
   * @returns {array} cluster names
   */
  getClusterNames() {
    return Object.getOwnPropertyNames(this.clusters)
  }

  /**
   * return an array of all hostnames for the given cluster
   * @param {string} clusterName name of the cluster to retrieve hosts for
   * @returns {array} host names
   */
  getHostNames(clusterName) {
    return Object.getOwnPropertyNames(this.clusters[clusterName])
  }

  /**
   * return an array of all available stats for the given cluster and host
   * @param {string} clusterName name of the cluster to retrieve hosts for
   * @param {string} statNamespace grouping of stats (e.g. host name) to retrieve stat names for
   * @returns {array} stat names
   */
  getStatNames(clusterName, statNamespace) {
    return Object.getOwnPropertyNames(this.clusters[clusterName][statNamespace])
  }

  /**
   * return the raw circular buffer for a stat
   * @param {string} clusterName name of the cluster to retrieve hosts for
   * @param {string} statNamespace grouping of stats (e.g. host name) to retrieve stat names for
   * @param {string} statName the actual stat to retrieve
   * @returns {array} raw stats
   */
  getStat(clusterName, statNamespace, statName) {
    if (this.clusters[clusterName] &&
        this.clusters[clusterName][statNamespace] &&
        this.clusters[clusterName][statNamespace][statName]) {
      return this.clusters[clusterName][statNamespace][statName]
    }
    this.log.error(`unknown series ${clusterName}::${statNamespace}${statName}`)
    return `err - ${clusterName}::${statNamespace}${statName}`
  }

  /**
   * compute deltas over the raw circular buffer, returning a metric series.
   * will return an object containing an x array (the metric value) and y array
   * (the textual timestamps of the metric values)
   * @param {string} clusterName name of the cluster to retrieve hosts for
   * @param {string} statNamespace grouping of stats (e.g. host name) to retrieve stat names for
   * @param {string} statName the actual stat to retrieve
   * @returns {array} array of delta metrics
   */
  getSeries(clusterName, statNamespace, statName) {
    let series = this.clusters[clusterName][statNamespace][statName]
    if (!series) {
      this.log.error(`unknown series ${clusterName}::${statNamespace}${statName}`)
      return null
    }
    this.log.debug(`series = ${series}`)
    let x = []
    let y = []
    let numSamples = 0
    for (let i = 2; i < this.bufferSize; i++) {
      let idx = (i + this.bufferIdx) % this.bufferSize
      if (typeof series[idx] !== 'undefined') {
        let lastIdx = idx - 1
        if (lastIdx < 0) {
          lastIdx = this.bufferSize - 1
        }
        let delta = series[idx] - series[lastIdx]
        if (!isNaN(delta)) {
          numSamples++
          y.push(series[idx] - series[lastIdx])
          x.push(this.times[idx])
        }
      }
    }
    if (numSamples > 0) {
      return {
        x: x,
        y: y,
      }
    }
    return null
  }

  /**
   * returning a gauge series for the given stat.
   * will return an object containing an x array (the metric value) and y array
   * (the textual timestamps of the metric values)
   * @param {string} clusterName name of the cluster to retrieve hosts for
   * @param {string} statNamespace grouping of stats (e.g. host name) to retrieve stat names for
   * @param {string} statName the actual stat to retrieve
   * @returns {array} gauge metrics
   */
  getSeriesAsGauge(clusterName, statNamespace, statName) {
    this.log.debug(`looking for ${statName}, bufferIdx=${this.bufferIdx}`)
    let series = this.clusters[clusterName][statNamespace][statName]
    if (!series) {
      this.log.error(`unknown series ${clusterName}::${statNamespace}${statName}`)
      return null
    }
    this.log.debug(`series = ${series}`)
    this.log.debug(`series = ${series}`)
    let x = []
    let y = []
    let numSamples = 0
    for (let i = 1; i < this.bufferSize; i++) {
      let idx = (i + this.bufferIdx) % this.bufferSize
      if (typeof series[idx] !== 'undefined') {
        numSamples++
        y.push(series[idx])
        x.push(this.times[idx])
      }
    }
    if (numSamples > 0) {
      return {
        x: x,
        y: y,
      }
    }
    return null
  }

  /**
   * stat a timer to poll the cluster endpoint on the given interval
   * @returns {null} nothing
   */
  start() {
    this.pollStats()
    setInterval(this.pollStats.bind(this), this.pollingInterval)
  }

  /**
   * call Envoy's <manager>/clusters endpoint, update metrics and stat names
   * @returns {null} nothing
   */
  pollStats() {
    http.get(this.clustersURI, res => {
      let body = '';
      res.on('data', data => {
        body = body + data;
      });
      res.on('end', () => {
        this.bufferIdx++
        this.bufferIdx = this.bufferIdx % this.bufferSize

        let now = dateFormat(new Date(), 'HH:MM:ss')
        this.times[this.bufferIdx] = now

        body.split('\n').forEach(m => {
          let splits = m.split('::')
          if (splits.length === 4) {
            let clusterName = splits[0]
            let statNamespace = splits[1]
            let statName = splits[2]
            let statValue = splits[3]
            let statNumeric = parseInt(statValue)
            if (typeof this.clusters[clusterName] === 'undefined') {
              this.clusters[clusterName] = {}
            }
            if (typeof this.clusters[clusterName][statNamespace] === 'undefined') {
              this.clusters[clusterName][statNamespace] = {}
            }
            if (!isNaN(statNumeric)) {
              if (typeof this.clusters[clusterName][statNamespace][statName] === 'undefined') {
                this.clusters[clusterName][statNamespace][statName] = []
              }
              this.clusters[clusterName][statNamespace][statName][this.bufferIdx] = statNumeric
            } else {
              this.clusters[clusterName][statNamespace][statName] = statValue
            }
          }
        })
        this.emit('updated')
      })
    })
  }
}

module.exports = Clusters
