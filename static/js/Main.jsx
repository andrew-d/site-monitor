/** @jsx React.DOM */

var React   = require('react'),
    Fluxxor = require('fluxxor'),
    _       = require('lodash');

var FluxMixin = Fluxxor.FluxMixin(React),
    FluxChildMixin = Fluxxor.FluxChildMixin(React),
    StoreWatchMixin = Fluxxor.StoreWatchMixin;

var ItemRow = require('./ItemRow.jsx'),
    NewItemForm = require('./NewItemForm.jsx');

var Main = React.createClass({
    mixins: [FluxMixin, StoreWatchMixin("ItemStore")],

    getStateFromFlux: function() {
        return this.getFlux().store("ItemStore").getState();
    },

    render: function() {
        var sortedItems = _.sortBy(this.state.items, 'seen');
        var itemRows = _.map(this.state.items, function(item, i) {
            return <ItemRow key={i} item={item} />
        });

        return (
            <div>
                <h2>Main</h2>

                <table className="table">
                    <thead>
                        <tr>
                            <th>Status</th>
                            <th>URL</th>
                            <th>Selector</th>
                            <th>Schedule</th>
                            <th>Last Successful Check</th>
                            <th>Hash</th>
                            <th>Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        {itemRows}
                    </tbody>
                </table>

                <NewItemForm />
            </div>
        );
    },
});

module.exports = Main;
