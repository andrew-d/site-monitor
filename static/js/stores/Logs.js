var Fluxxor = require('fluxxor'),
    moment  = require('moment'),
    request = require('superagent');

var LogsStore = Fluxxor.createStore({
    actions: {
        'ADD_LOG': 'onAddLog',
        'SERVER_LOG': 'onServerLog',
        'CLEAR_LOGS': 'onClearLogs',
        'REFRESH_LOGS': 'onRefreshLogs',
    },

    initialize: function() {
        this.logs = [];
        this.onRefreshLogs();
    },

    onAddLog: function(payload) {
        this.logs.push({
            time:    moment().format("YYYY-MM-DDTHH:mm:ssZ"),
            level:   payload.level,
            message: payload.message,
            fields:  {},
        });
        this.emit('change');
    },

    onServerLog: function(payload) {
        this.logs.push(payload);
        this.emit('change');
    },

    onClearLogs: function() {
        request
            .del("/api/logs")
            .end(function(res) {
                this.logs = [];
                this.emit('change');
            }.bind(this));
    },

    onRefreshLogs: function() {
        request
            .get('/api/logs')
            .type('json')
            .set('Accept', 'application/json')
            .end(function(res) {
                // TODO: error checking.
                // TODO: merge log entries
                this.logs = this.logs.concat(res.body);
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
    serverLogNotification: function(logdata) {
        this.dispatch("SERVER_LOG", logdata);
    },
};

module.exports = {
    actions: actions,
    LogsStore: LogsStore,
};
