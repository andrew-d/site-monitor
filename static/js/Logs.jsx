/** @jsx React.DOM */

'use strict';

var React   = require('react'),
    Fluxxor = require('fluxxor'),
    _       = require('lodash');

var FluxMixin = Fluxxor.FluxMixin(React),
    StoreWatchMixin = Fluxxor.StoreWatchMixin;

var LogRow = require('./LogRow.jsx');

var Logs = React.createClass({
    mixins: [FluxMixin, StoreWatchMixin("LogStore")],

    componentDidMount: function() {
        // Re-render every minute to update the times in the log rows.
        this.timeoutId = window.setTimeout(function() {
            this.forceUpdate();
        }.bind(this), 60*1000);
    },

    componentWillUnmount: function() {
        window.clearTimeout(this.timeoutId);
    },

    getStateFromFlux: function() {
        return this.getFlux().store("LogStore").getState();
    },

    render: function() {
        var sortedLogs = this.state.logs.slice(0);
        sortedLogs.sort(function(a, b) {
            if( a.time === b.time ) {
                return 0;
            } else if( a.time < b.time ) {
                return 1;
            } else {
                return -1;
            }
        });

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
                    {_.map(sortedLogs, function(log, i) {
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
