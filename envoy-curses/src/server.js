/* eslint camelcase: ["error", {properties: "never"}]*/

let blessed = require('blessed')
let Theme = require('./theme')
let Box = blessed.Box
let contrib = require('blessed-contrib')
let Menu = require('./menu.js')

class Server extends Box {
  constructor(options) {
    super(options)
    this.options = options || {};

    this.style = this.options.style || Theme.style.base

    this.domain = options.domain
    this.screen = options.screen
    this.refresh = false
    this.stats = options.stats
    this.log = options.log

    this.gauges = [
      {
        label: 'Uptime',
        stat: 'server.uptime',
      },
      {
        label: 'Version',
        stat: 'server.version',
      },
      {
        label: 'Watchdog Miss',
        stat: 'server.watchdog_miss',
      },
      {
        label: 'Watchdog Mega Miss',
        stat: 'server.watchdog_mega_miss',
      },
      {
        label: 'Memory Allocated',
        stat: 'server.memory_allocated',
      },
      {
        label: 'Parent Connections',
        stat: 'server.parent_connections',
      },
      {
        label: 'Memory Heap Size',
        stat: 'server.memory_heap_size',
      },
      {
        label: 'Total Connections',
        stat: 'server.total_connections',
      },
    ]


    this.memorySeries = [
      {
        title: 'Allocated',
        stat_name: 'server.memory_allocated',
        style: {line: Theme.pickChartColor(0, 2)},
        x: [],
        y: [],
      },
      {
        title: 'Heap Size',
        stat_name: 'server.memory_heap_size',
        style: {line: Theme.pickChartColor(1, 2)},
        x: [],
        y: [],
      },
    ]
    this.memoryLine = contrib.line(
      {
        label: 'MemoryUse',
        showLegend: true,
        top: Math.floor(this.gauges.length/2 + 3),
        left: 0,
        width: '50%',
        legend: {
          width: 20,
        },
        style: Theme.style.chart,
      })

    this.connectionsSeries = [
      {
        title: 'Parent Connections',
        stat_name: 'server.parent_connections',
        style: {line: Theme.pickChartColor(0, 2)},
        x: [],
        y: [],
      },
      {
        title: 'Total Connections',
        stat_name: 'server.total_connections',
        style: {line: Theme.pickChartColor(1, 2)},
        x: [],
        y: [],
      },
    ]
    this.connectionsLine = contrib.line(
      {
        label: 'Connections',
        showLegend: true,
        top: Math.floor(this.gauges.length/2 + 3),
        left: '50%',
        width: '50%',
        legend: {
          width: 20,
        },
        style: Theme.style.chart,
      })

    this.stats.on('updated', () => {
      this.gauges.forEach(g => {
        g.target.content = '' + this.stats.getCurrentStatValue(g.stat)
      })
      this.memorySeries.forEach(s => {
        this.log.debug(`getting stat ${s.stat_name}`)
        let currentSeries = this.stats.getSeriesAsGauge(s.stat_name)
        if (currentSeries) {
          s.x = currentSeries.x
          s.y = currentSeries.y
        } else {
          this.log.debug("couldn't find series")
        }
      })
      this.connectionsSeries.forEach(s => {
        this.log.debug(`getting connections stat ${s.stat_name}`)
        let currentSeries = this.stats.getSeriesAsGauge(s.stat_name)
        if (currentSeries) {
          s.x = currentSeries.x
          s.y = currentSeries.y
        } else {
          this.log.debug(`could not find series ${s.stat_name} - ${currentSeries}`)
        }
      })
      if (this.parent) {
        this.connectionsLine.setData(this.connectionsSeries);
        this.memoryLine.setData(this.memorySeries);
        this.screen.render()
      }
    })

    this.appendGauges = () => {
      for (let i = 0; i < this.gauges.length; i++) {
        let label = Box({
          top: Math.floor(i/2) + 3,
          left: `${50*(i%2)}%`,
          height: 1,
          width: '25%',
          style: Theme.style.nofocus,
          content: this.gauges[i].label,
        })
        let target = Box({
          top: Math.floor(i/2) + 3,
          left: `${50*(i%2) + 25}%`,
          height: 1,
          width: '25%',
          style: Theme.style.nofocus,
          content: 'tbd',
        })
        this.gauges[i].target = target
        this.append(label)
        this.append(target)
      }
    }
  }

  show(screen) {
    this.append(new Menu({
      screen: this.screen,
      selected: 'Server',
    }))
    this.appendGauges()
    this.append(this.memoryLine)
    this.append(this.connectionsLine)
    screen.append(this)
  }
}

module.exports = Server
