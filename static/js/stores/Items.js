var Fluxxor = require('fluxxor'),
    request = require('superagent'),
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
        request
            .post('/api/checks')
            .type('json')
            .accept('json')
            .send({
                url:      payload.url,
                selector: payload.selector,
                schedule: payload.schedule,
            })
            .end(function(res) {
                // TODO: error checking
                this.items.push(res.body);
                this.emit('change');
            }.bind(this));
    },

    onRefreshItems: function() {
        request
            .get('/api/checks')
            .type('json')
            .accept('json')
            .end(function(res) {
                // TODO: error checking
                this.items = res.body;
                this.emit('change');
            }.bind(this));
    },

    onDeleteItem: function(id) {
        if( this._hasItem(id) ) {
            request
                .del('/api/checks/' + id)
                .end(function(res) {
                    // TODO: error checking
                    this.items = _.reject(this.items, {'id': id});
                    this.emit('change');
                }.bind(this));
        }
    },

    onMarkItemRead: function(id) {
        var item = _.find(this.items, {'id': id});
        if( item ) {
            request
                .patch('/api/checks/' + id)
                .type('json')
                .accept('json')
                .send({seen: true})
                .end(function(res) {
                    // TODO: error checking
                    item.seen = true;
                    this.emit('change');
                }.bind(this));
        }
    },

    onRefreshItem: function(id) {
        // TODO: talk to server
    },


    getState: function() {
        return {
            items: this.items,
        };
    },

    _hasItem: function(id) {
        return _.find(this.items, {'id': id}) !== undefined;
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
