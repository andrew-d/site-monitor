var Fluxxor = require('fluxxor'),
    $       = require('jquery');

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
        this.logs.push({
            level:   payload.level,
            message: payload.message,
        });
        this.emit('change');
    },

    onClearLogs: function() {
        this.logs = [];
        this.emit('change');
    },

    onRefreshLogs: function() {
        var self = this;

        $.getJSON('/api/logs', function(data) {
            // TODO: should merge intelligently
            self.logs.concat(data);

            if( data.length > 0 ) {
                self.emit('change');
            }
        });
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
};

module.exports = {
    actions: actions,
    LogsStore: LogsStore,
};
