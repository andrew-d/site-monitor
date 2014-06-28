var Fluxxor = require('fluxxor'),
    $       = require('jquery'),
    _       = require('lodash');

var ItemsStore = Fluxxor.createStore({
    actions: {
        'ADD_ITEM': 'onAddItem',
        'REFRESH_ITEMS': 'onRefreshItems',
        "DELETE_ITEM": 'onDeleteItem',
        "MARK_ITEM_READ": 'onMarkItemRead',
        "REFRESH_ITEM": 'onRefreshItem',
    },

    initialize: function() {
        this.items = [];
        this.onRefreshItems();
    },

    onAddItem: function(payload) {
        this.items.push({
            url:      payload.url,
            selector: payload.selector,
            schedule: payload.schedule,
        });
        // TODO: save to server
        this.emit('change');
    },

    onRefreshItems: function() {
        var self = this;

        $.getJSON("/api/checks", function(data) {
            self.items = data;
            self.emit('change');
        });
    },

    onDeleteItem: function(id) {
        this.items = _.reject(this.items, {'id': id});
        // TODO: delete on server
        this.emit('change');
    },

    onMarkItemRead: function(id) {
        // TODO: talk to server
    },

    onRefreshItem: function(id) {
        // TODO: talk to server
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

    refreshItems: function() {
        this.dispatch("REFRESH_ITEMS");
    },

    deleteItem: function(id) {
        this.dispatch("DELETE_ITEM", id);
    },

    markItemRead: function(id) {
        this.dispatch("MARK_ITEM_READ", id);
    },

    refreshItem: function(id) {
        this.dispatch("REFRESH_ITEM", id);
    },
};

module.exports = {
    actions: actions,
    ItemsStore: ItemsStore,
};
