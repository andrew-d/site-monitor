/** @jsx React.DOM */

'use strict';

var React = require('react'),
    Fluxxor = require('fluxxor');

var FluxMixin = Fluxxor.FluxMixin(React),
    StoreWatchMixin = Fluxxor.StoreWatchMixin;

var LogRow = require('./LogRow.jsx');

var Logs = React.createClass({
    mixins: [FluxMixin, StoreWatchMixin("LogStore")],

    getStateFromFlux: function() {
        return this.getFlux().store("LogStore").getState();
    },

    render: function() {
        return (
            <div>
                <h2>Logs</h2>
                <button onClick={this.onClick} className="btn btn-primary">Clear Logs</button>

                <table className="table">
                    <thead>
                        <tr>
                            <th>Time</th>
                            <th>Level</th>
                            <th>Message</th>
                            <th>Fields</th>
                        </tr>
                    </thead>

                    <tbody>
                    {this.state.logs.map(function(log, i) {
                        return <LogRow key={i} log={log} />
                    })}
                    </tbody>
                </table>
            </div>
        );
    },

    onClick: function() {
        this.getFlux().actions.clearLogs();
    },
});

module.exports = Logs;
