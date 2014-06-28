var Fluxxor = require('fluxxor'),
    moment  = require('moment'),
    request = require('superagent');

var LogsStore = Fluxxor.createStore({
    actions: {
        'ADD_LOG': 'onAddLog',
        'CLEAR_LOGS': 'onClearLogs',
        'REFRESH_LOGS': 'onRefreshLogs',
    },

    initialize: function() {
        this.logs = [];
        this.onRefreshLogs();
    },

    onAddLog: function(payload) {
        var now = moment();
        this.logs.push({
            time:    now.format("YYYY-MM-DDTHH:mm:ssZ"),
            level:   payload.level,
            message: payload.message,
            fields:  {},

            'moment':  now,
        });
        this.emit('change');
    },

    onClearLogs: function() {
        this.logs = [];
        this.emit('change');
    },

    onRefreshLogs: function() {
        request
            .get('/api/logs')
            .type('json')
            .set('Accept', 'application/json')
            .end(function(res) {
                // TODO: error checking.
                this.logs = this.logs.concat(_.map(res.body, this._extendLog));
                if( res.body.length > 0 ) {
                    this.emit('change');
                }
            }.bind(this));
    },

    getState: function() {
        return {
            logs: this.logs,
        };
    },

    _extendLog: function(log) {
        var ptime = moment(log.time, "YYYY-MM-DDTHH:mm:ssZ");
        if( !ptime.isValid() ) {
            return log;
        }

        return _.extend(_.clone(log), {"moment": ptime});
    },
});


// TODO: set up websocket connection or something to update this


var actions = {
    addLog: function(level, message) {
        this.dispatch("ADD_LOG", {
            level:   level,
            message: message,
        });
    },
    clearLogs: function() {
        this.dispatch("CLEAR_LOGS");
    },
    refreshLogs: function() {
        this.dispatch("REFRESH_LOGS");
    },
};

module.exports = {
    actions: actions,
    LogsStore: LogsStore,
};
