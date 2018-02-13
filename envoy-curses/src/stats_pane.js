let blessed = require('blessed')
let Theme = require('./theme')
let Box = blessed.Box
let contrib = require('blessed-contrib')
let Menu = require('./menu.js')
let ListTable = blessed.ListTable
let theme = require('./theme.js')

class StatsPane extends Box {
  constructor(options) {
    super(options)
    this.options = options || {}
    this.screen = options.screen
    this.style = this.options.style || Theme.style.base
    this.stats = options.stats
    this.log = options.log
    this.chartedStat = 'rq_total'
    this.availableStats = []
    this.selectedListenerName = ''
    this.statsFilter = null
    this.treeData = {}
    this.legendWidth = 20

    /* eslint camelcase: ["error", {properties: "never"}]*/
    this.reservedHostnames = {
      default_priority: true,
      high_priority: true,
      added_via_api: true,
    }

    this.statsSearch = Box({
      label: 'Search',
      content: '',
      top: 3,
      height: 3,
      fg: Theme.style.base.fg,
      border: {type: 'line', fg: Theme.style.base.border.fg},
      width: '100%',
      keys: true,
      mouse: true,
    })

    this.statsList = ListTable({
      interactive: true,
      align: 'left',
      width: '50%-1',
      top: 7,
      height: '100%-7',
      border: {type: 'line', fg: Theme.style.table.border},
      style: {
        cell: {
          item: {
            fg: theme.style.list.item.fg,
            bg: theme.style.list.item.bg,
          },
          selected: {
            fg: theme.style.list.selected.fg,
            bg: theme.style.list.selected.bg,
          },
        },
      },
    })

    this.connectionsLine = contrib.line(
      {
        label: 'Stats',
        showLegend: true,
        top: 7,
        left: '50%',
        width: '50%',
        height: '100%-7',
        border: {type: 'line', fg: Theme.style.table.border},
        legend: {width: 40},
        style: Theme.style.chart,
      })

    this.connectionsSeries = null

    this.statsList.on('select item', (selected, idx) => {
      if (selected.content) {
        this.chartedStat = selected.content.split(/\s+/)[0]
        this.updateChartData()
        this.updateView()
      }
    })

    this.on('attach', () => {
      this.statsSearch.focus()
    })

    this.statsSearch.on('keypress', (k, d) => {
      if (d.full.length === 1) {
        this.statsSearch.content = this.statsSearch.content + d.full
        this.updateStatNames()
        this.statsList.select(1)
      } else {
        let content = this.statsSearch.content
        if (d.name === 'backspace') {
          content = content.substr(0, content.length - 1)
          this.statsSearch.content = content
        } else if (d.name === 'down' || d.full === 'C-n') {
          this.statsList.down()
        } else if (d.name === 'up' || d.full === 'C-t') {
          this.statsList.up()
        } else if (d.name === 'escape') {
          this.statsSearch.content = ''
          this.updateStatNames()
          this.statsList.select(1)
        }
      }
      this.screen.render()
    })

    this.stats.on('updated', () => {
      if (this.parent) {
        this.updateStatNames()
        this.updateChartData()
        this.updateView()
        this.screen.render()
      }
    })
  }

  updateStatNames() {
    let selected = this.statsList.selected
    let st = this.stats.getStatsTable(new RegExp(`.*${this.statsSearch.content}.*`))
    this.statsList.setData(st)
    if (selected < this.statsList.items.length) {
      this.statsList.select(selected)
    }
  }

  updateChartData() {
    if (this.chartedStat) {
      let seriesData = this.stats.getSeries(this.chartedStat)
      let title = this.chartedStat
      if (title.length > this.legendWidth) {
        title = `...${title.substring(title.length - this.legendWidth - 3)}`
      }
      if (seriesData) {
        this.connectionsSeries = [{
          title: title,
          stat_name: this.chartedStat,
          style: {
            line: theme.pickChartColor(0, 10),
          },
          x: seriesData.x,
          y: seriesData.y,
        }]
      }
    }
  }

  updateView() {
    if (this.parent) {
      if (this.connectionsSeries) {
        this.connectionsLine.setData(this.connectionsSeries)
        this.connectionsLine.setLabel(`${this.chartedStat}`)
      }
    }
  }


  show(screen) {
    this.append(new Menu({
      screen: screen,
      selected: 'Stats',
    }))
    screen.append(this)
    this.append(this.statsSearch)
    this.append(this.statsList)
    this.append(this.connectionsLine)
    this.updateView()
    this.updateChartData()
    this.updateStatNames()
    this.statsSearch.focus()
  }
}

module.exports = StatsPane
