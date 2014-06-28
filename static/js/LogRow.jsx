/** @jsx React.DOM */

'use strict';

var React  = require('react'),
    _      = require('lodash');

var LogRow = React.createClass({
    propTypes: {
        log: React.PropTypes.shape({
            time:    React.PropTypes.string.isRequired,
            level:   React.PropTypes.string.isRequired,
            message: React.PropTypes.string.isRequired,
            fields:  React.PropTypes.object.isRequired,

            moment:  React.PropTypes.object,
        }),
    },

    render: function() {
        var time;

        if( _.has(this.props.log, 'moment') ) {
            time = (
                <span title={this.props.log.time}>
                    {this.props.log.moment.fromNow()}
                </span>
            );
        } else {
            time = (
                <span title="(invalid format)">
                    {this.props.log.time}
                </span>
            );
        }

        var fields = _.map(this.props.log.fields, function(val, key) {
            return <div key={key}><b>{key}: </b>{val}</div>
        });

        return (
            <tr>
                <td>{time}</td>
                <td>{this.props.log.level}</td>
                <td>{this.props.log.message}</td>
                <td>{fields}</td>
            </tr>
        );
    },
});

module.exports = LogRow;
