/** @jsx React.DOM */

'use strict';

var React  = require('react'),
    moment = require('moment'),
    _      = require('lodash');

var LogRow = React.createClass({
    propTypes: {
        log: React.PropTypes.shape({
            time:    React.PropTypes.string.isRequired,
            level:   React.PropTypes.string.isRequired,
            message: React.PropTypes.string.isRequired,
            fields:  React.PropTypes.object.isRequired,
        }),
    },

    render: function() {
        var time;

        var ptime = moment(this.props.log.time, "YYYY-MM-DDTHH:mm:ssZ");
        if( !ptime.isValid() ) {
            time = (
                <span title="(invalid format)">
                    {this.props.log.time}
                </span>
            );
        } else {
            time = (
                <span title={this.props.log.time}>
                    {ptime.fromNow()}
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
