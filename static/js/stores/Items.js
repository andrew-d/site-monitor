var Fluxxor = require('fluxxor'),
    request = require('superagent'),
    _       = require('lodash');

var ItemsStore = Fluxxor.createStore({
    actions: {
        'ADD_ITEM': 'onAddItem',
        "SERVER_ITEM": "onServerItem",
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

    onServerItem: function(payload) {
        var oldItem = _.find(this.items, {'id': payload.id});
        if( !oldItem ) {
            this.items.push(payload);
            this.emit('change');
            return;
        }

        // Merge the two objects, keeping track whether they changed or not.
        var changed = false;
        _.each(_.keys(payload), function(key) {
            if( oldItem[key] !== payload[key] ) {
                changed = true;
            }

            oldItem[key] = payload[key];
        });

        if( changed ) {
            this.emit('change');
        }
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
        if( this._hasItem(id) ) {
            request
                .post('/api/checks/' + id + '/update')
                .accept('json')
                .end(function(res) {
                    // TODO: error checking
                    _.assign(_.find(this.items, {'id': id}), res.body);
                    this.emit('change');
                }.bind(this));
        }
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

    serverItemNotification: function(item) {
        this.dispatch("SERVER_ITEM", item);
    },
};

module.exports = {
    actions: actions,
    ItemsStore: ItemsStore,
};
