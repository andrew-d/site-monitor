var Fluxxor = require('fluxxor'),
    _       = require('underscore');

var Items = require('./Items.js'),
    Logs  = require('./Logs.js');


var stores = {
    ItemStore: new Items.ItemsStore(),
    LogStore: new Logs.LogsStore(),
};

var actions = _.extend({}, Items.actions, Logs.actions);

module.exports = window.flux = new Fluxxor.Flux(stores, actions);
