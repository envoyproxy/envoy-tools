/* eslint camelcase: ["error", {properties: "never"}]*/

const blessed = require('blessed')
const Theme = require('./theme')
const Box = blessed.Box
const contrib = require('blessed-contrib')
const Menu = require('./menu.js')
const theme = require('./theme.js')

class ClustersPane extends Box {
  constructor(options) {
    super(options)
    this.options = options || {}
    this.screen = options.screen
    this.style = this.options.style || Theme.style.base
    this.clusters = options.clusters
    this.stats = options.stats
    this.log = options.log
    this.chartedStat = 'rq_total'
    this.availableStats = []
    this.selectedClusterName = ''
    this.tableData = {
      headers: ['cluster', 'cx act', 'rq act', 'rq total', 'members', 'healthy'],
      data: [],
    }

    this.connectionsSeries = []

    this.clusterLevelStats = new Set()
    this.clusterLevelStats.add('default_priority')
    this.clusterLevelStats.add('high_priority')
    this.clusterLevelStats.add('added_via_api')

    this.clustersTable = contrib.table({
      fg: Theme.style.table.fg,
      selectedFg: Theme.style.table.selectedFg,
      selectedBg: Theme.style.table.selectedBg,
      keys: true,
      interactive: true,
      label: 'Clusters',
      width: '50%',
      top: 3,
      height: '100%-3',
      border: { type: 'line', fg: Theme.style.table.border },
      columnSpacing: 2,
      columnWidth: [20, 8, 8, 8, 8, 8],
    })

    this.connectionsLine = contrib.line({
      label: 'Stats',
      showLegend: true,
      top: 3,
      left: '50%',
      width: '50%',
      height: '100%-3',
      border: { type: 'line', fg: Theme.style.table.border },
      legend: { width: 20 },
      style: Theme.style.chart,
    })

    const searchStyle = Object.assign(
      {
        item: {
          hover: {
            bg: Theme.style.base.fg,
          },
        },
        selected: {
          bg: Theme.style.base.focus.bg,
          fg: Theme.style.base.focus.fg,
          bold: true,
        },
      },
      Theme.style.base
    )

    this.statSearch = blessed.List({
      label: 'Stats',
      width: '50%',
      height: '50%',
      top: 'center',
      left: 'center',
      hidden: true,
      style: searchStyle,
      border: { type: 'line', fg: Theme.style.base.border.fg },
      keys: true,
      interactive: true,
    })

    /**
     * we've selected a new cluster, so update available stats, the underlying
     * chart model, and update the view
     */
    this.clustersTable.rows.on('select', cluster => {
      this.selectedClusterName = cluster.content.split(/\s+/)[0]
      this.updateAvailableStats()
      this.updateChartData()
      this.updateView()
    })

    /**
     * our cluster model is updated, so update the chart and table, and render the view
     */
    this.clusters.on('updated', () => {
      this.updateChartData()
      this.updateTableData()
      this.updateView()
    })

    /**
     * handle '/' and '?' to launch the stat selection
     */
    this.on('element keypress', (ch, key) => {
      if (!this.detached) {
        if (key === '/' || key === '?') {
          this.statSearch.focus()
          this.statSearch.show()
          this.screen.render()
          this.statSearch.once('action', (el, selected) => {
            this.statSearch.hide()
            if (el) {
              this.selectStat(el.content)
            }
            this.clustersTable.focus()
            this.updateChartData()
            this.updateView()
          })
        }
      }
    })

    this.on('attach', () => {
      this.clustersTable.focus()
    })
  }

  selectStat(s) {
    if (s) {
      this.chartedStat = s
      this.updateAvailableStats()
    }
  }

  /**
   * build a set of all available stats for the currently selected cluster
   * @returns {null} nothing
   */
  updateAvailableStats() {
    const hostNames = this.clusters.getHostNames(this.selectedClusterName)
    const newStats = new Set()
    for (let i = 0; i < hostNames.length; i++) {
      if (!this.clusterLevelStats.has(hostNames[i])) {
        this.clusters.getStatNames(this.selectedClusterName, hostNames[i]).forEach(s => {
          newStats.add(s)
        })
      }
    }
    this.statSearch.clearItems()
    this.availableStats = Array.from(newStats).sort()
    for (let i = 0; i < this.availableStats.length; i++) {
      this.statSearch.addItem(this.availableStats[i])
      if (this.availableStats[i] === this.chartedStat) {
        this.statSearch.select(i)
      }
    }
  }

  /**
   * given a list of cluster names, build the table data for them
   * @returns {null} nothing
   */
  updateTableData() {
    const clusterNames = this.clusters.getClusterNames()
    const newTableData = clusterNames.map(c => {
      const row = []
      if (!this.selectedClusterName) {
        this.selectedClusterName = c
        this.updateAvailableStats()
      }
      row.push(c)
      row.push(this.stats.getCurrentStatValue(`cluster.${c}.upstream_cx_active`))
      row.push(this.stats.getCurrentStatValue(`cluster.${c}.upstream_rq_active`))
      row.push(this.stats.getCurrentStatValue(`cluster.${c}.upstream_rq_total`))
      row.push(this.stats.getCurrentStatValue(`cluster.${c}.membership_total`))
      row.push(this.stats.getCurrentStatValue(`cluster.${c}.membership_healthy`))
      return row
    })
    this.tableData.data = newTableData
  }

  /**
   * update underlying data model for our chart. Does not actualy attach
   * the data model to the chart to prevent the chart trying to render when
   * we're not visible
   * @returns {null} nothing
   */
  updateChartData() {
    if (!this.selectedClusterName) {
      return
    }
    let hostNames = this.clusters.getHostNames(this.selectedClusterName)
    const series = []
    for (let i = 0; i < hostNames.length; i++) {
      if (!this.clusterLevelStats.has(hostNames[i])) {
        let currentSeries = this.clusters.getSeries(
          this.selectedClusterName,
          hostNames[i],
          this.chartedStat
        )
        if (currentSeries) {
          series.push({
            title: hostNames[i],
            cluster_name: this.selectedClusterName,
            stat_namespace: hostNames[i],
            stat_name: this.chartedStat,
            style: {
              line: theme.pickChartColor(i, hostNames.length),
            },
            x: currentSeries.x,
            y: currentSeries.y,
          })
        }
      }
    }
    this.connectionsSeries = series
    this.connectionsLine.setLabel(`${this.selectedClusterName} - ${this.chartedStat}`)
  }

  /**
   * actually attach data models to the chart and table, then render screen
   * @returns {null} nothing
   */
  updateView() {
    if (this.parent) {
      if (this.connectionsSeries && this.connectionsSeries.length > 0) {
        this.connectionsLine.setData(this.connectionsSeries)
      }
      this.clustersTable.setData(this.tableData)
      this.screen.render()
    }
  }

  show(screen) {
    this.append(
      new Menu({
        screen: screen,
        selected: 'Clusters',
      })
    )
    this.append(this.clustersTable)
    this.append(this.connectionsLine)
    this.append(this.statSearch)
    screen.append(this)
    this.updateTableData()
    this.updateAvailableStats()
    this.updateChartData()
    this.updateView()
  }
}

module.exports = ClustersPane
