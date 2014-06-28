/** @jsx React.DOM */

'use strict';

var React = require('react'),
    Fluxxor = require('fluxxor');

var FluxChildMixin = Fluxxor.FluxChildMixin(React);

var ActionButton = require('./ActionButton.jsx');

var ItemRow = React.createClass({
    mixins: [FluxChildMixin],

    propTypes: {
        item: React.PropTypes.shape({
            id:       React.PropTypes.number.isRequired,
            url:      React.PropTypes.string.isRequired,
            selector: React.PropTypes.string.isRequired,
            schedule: React.PropTypes.string.isRequired,
            seen:     React.PropTypes.bool.isRequired,
        }),
    },

    render: function() {
        var label;
        if( this.props.item.seen ) {
            label = <span className="label label-default">Seen</span>;
        } else {
            label = <span className="label label-primary">Changed</span>;
        }

        return (
            <tr>
                <td>{label}</td>
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
