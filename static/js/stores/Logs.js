var Fluxxor = require('fluxxor');

var LogsStore = Fluxxor.createStore({
    actions: {
        'ADD_LOG': 'onAddLog',
        'CLEAR_LOGS': 'onClearLogs',
    },

    initialize: function() {
        this.logs = [];
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

    getState: function() {
        return {
            logs: this.logs,
        };
    },
});


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
};

module.exports = {
    actions: actions,
    LogsStore: LogsStore,
};
