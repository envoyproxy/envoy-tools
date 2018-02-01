const blessed = require('blessed')
const Carousel = require('blessed-contrib').carousel
const log = require('simple-node-logger').createSimpleFileLogger('envoy-curses.log')
const Stats = require('./stats.js')
const StatsPane = require('./stats_pane.js')
const Clusters = require('./clusters.js')
const ClustersPane = require('./clusters_pane.js')
const Server = require('./server.js')

const screen = blessed.screen()
const adminServerAddress = process.argv[2] || 'http://localhost:9000'
const pollingInterval = parseInt(process.argv[3]) || 1000

log.setLevel('info')

// create layout and widgets

const stats = new Stats({
  log: log,
  pollingInterval: pollingInterval,
  statsURI: `${adminServerAddress}/stats`,
})

const clusters = new Clusters({
  log: log,
  pollingInterval: pollingInterval,
  clustersURI: `${adminServerAddress}/clusters`,
})

const statsPane = new StatsPane({
  domain: 'http.gke-proxy-80',
  stats: stats,
  screen: screen,
  log: log,
})

const clustersPane = new ClustersPane({
  clusters: clusters,
  stats: stats,
  screen: screen,
  log: log,
})

const server = new Server({
  stats: stats,
  screen: screen,
  log: log,
})

const carousel = new Carousel(
  [server.show.bind(server), clustersPane.show.bind(clustersPane), statsPane.show.bind(statsPane)],
  {
    screen: screen,
    interval: 0,
    controlKeys: true,
  }
)

stats.start()
clusters.start()

screen.key(['C-c', 'C-d'], (ch, key) => {
  return process.exit(0)
})

carousel.start()
