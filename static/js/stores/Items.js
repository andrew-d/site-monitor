var Fluxxor = require('fluxxor');

var ItemsStore = Fluxxor.createStore({
    actions: {
        'ADD_ITEM': 'onAddItem',
    },

    initialize: function() {
        this.items = [];
    },

    onAddItem: function(payload) {
        this.items.push({
            url:      payload.url,
            selector: payload.selector,
            schedule: payload.schedule,
        });
        this.emit('change');
    },

    getState: function() {
        return {
            items: this.items,
        };
    },
});

var actions = {
    addItem: function(url, selector, schedule) {
        this.dispatch("ADD_ITEM", {
            url: url,
            selector: selector,
            schedule: schedule,
        });
    },
};

module.exports = {
    actions: actions,
    ItemsStore: ItemsStore,
};
