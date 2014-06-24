/** @jsx React.DOM */

'use strict';

var React = require('react');

var LogRow = React.createClass({
    propTypes: {
        log: React.PropTypes.shape({
            level:   React.PropTypes.string.isRequired,
            message: React.PropTypes.string.isRequired,
        }),
    },

    render: function() {
        return (
            <tr>
                <td>Time</td>
                <td>{this.props.log.level}</td>
                <td>{this.props.log.message}</td>
                <td>Fields</td>
            </tr>
        );
    },
});

module.exports = LogRow;
