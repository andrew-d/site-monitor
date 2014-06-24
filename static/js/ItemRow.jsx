/** @jsx React.DOM */

'use strict';

var React = require('react'),
    ActionButton = require('./ActionButton.jsx');

var ItemRow = React.createClass({
    propTypes: {
        item: React.PropTypes.shape({
            url:      React.PropTypes.string.isRequired,
            selector: React.PropTypes.string.isRequired,
            schedule: React.PropTypes.string.isRequired,
        }),
    },

    render: function() {
        return (
            <tr>
                <td>Label</td>
                <td>{this.props.item.url}</td>
                <td>{this.props.item.selector}</td>
                <td>{this.props.item.schedule}</td>
                <td>Last Successful Check</td>
                <td>Hash</td>
                <td>
                    <ActionButton
                        type="btn-success"
                        icon="glyphicon-ok"
                        action={this.onMarkRead} />
                    <ActionButton
                        type="btn-danger"
                        icon="glyphicon-remove"
                        action={this.onDeleteItem} />
                    <ActionButton
                        type="btn-primary"
                        icon="glyphicon-refresh"
                        action={this.onRefreshItem} />
                </td>
            </tr>
        );
    },

    onMarkRead: function() {
        console.log("Marking as read");
    },

    onDeleteItem: function() {
        console.log("Deleting item");
    },

    onRefreshItem: function() {
        console.log("Refreshing item");
    },
});

module.exports = ItemRow;
