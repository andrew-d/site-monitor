package main

import (
	"fmt"
	"io"

	"github.com/hoisie/mustache"
)

var templates map[string]*mustache.Template

func ParseTemplate(name, contents string) {
	tmpl, err := mustache.ParseString(contents)
	if err != nil {
		panic(err)
	}

	templates[name] = tmpl
}

func RenderTemplate(name string, context ...interface{}) string {
	layoutTemplate := templates["layout"]
	res := templates[name].RenderInLayout(layoutTemplate, context...)
	return res
}

func RenderTemplateTo(w io.Writer, name string, context ...interface{}) {
	_, err := fmt.Fprint(w, RenderTemplate(name, context...))
	if err != nil {
		panic(err)
	}
}

func init() {
	templates = make(map[string]*mustache.Template)

	ParseTemplate("layout", `
	<!DOCTYPE html>
	<html lang="en">
		<head>
			<meta charset="utf-8">
			<meta http-equiv="X-UA-Compatible" content="IE=edge">
			<meta name="viewport" content="width=device-width, initial-scale=1">
			<title>Checker</title>

			<link href="/static/css/bootstrap.min.css" rel="stylesheet">

			<!-- HTML5 Shim and Respond.js allow IE8 to support HTML5 elements and media queries -->
			<!--[if lt IE 9]>
				<script src="https://oss.maxcdn.com/libs/html5shiv/3.7.0/html5shiv.js"></script>
				<script src="https://oss.maxcdn.com/libs/respond.js/1.4.2/respond.min.js"></script>
			<![endif]-->

			<script src="https://ajax.googleapis.com/ajax/libs/jquery/1.11.0/jquery.min.js"></script>
		</head>
		<body>
			<div class="navbar navbar-default navbar-static-top" role="navigation">
				<div class="container">
					<div class="navbar-header">
						<button type="button" class="navbar-toggle" data-toggle="collapse" data-target=".navbar-collapse">
							<span class="sr-only">Toggle navigation</span>
							<span class="icon-bar"></span>
							<span class="icon-bar"></span>
							<span class="icon-bar"></span>
						</button>
						<a class="navbar-brand" href="/">Checker</a>
					</div>
					<div class="navbar-collapse collapse">
						<ul class="nav navbar-nav">
							<li><a href="/">Home</a></li>
							<li><a href="/about">About</a></li>
						</ul>
					</div>
				</div>
			</div>

			<div class="container">
				{{{content}}}
			</div>

			<script src="/static/js/bootstrap.min.js"></script>
		</body>
	</html>
	`)

	ParseTemplate("index", `
	<h1>Checks</h1>

	<table class="table">
		<thead>
			<tr>
				<th>Status</th>
				<th>URL</th>
				<th>Selector</th>
				<th>Schedule</th>
				<th>Last Checked</th>
				<th>Hash</th>
				<th>Actions</th>
			</tr>
		</thead>
		{{#items}}
			<tr>
				<td>
				{{#SeenChange}}
					<span class="label label-default">OK</span>
				{{/SeenChange}}
				{{^SeenChange}}
					<span class="label label-primary">Changed</span>
				{{/SeenChange}}
				</td>
				<td>{{URL}}</td>
				<td>{{Selector}}</td>
				<td>{{Schedule}}</td>
				<td>{{LastCheckedPretty}}</td>
				<td>
					<span data-hash="{{LastHash}}">
						{{ShortHash}}
					</span>
				</td>
				<td>
					<form style="display: inline" action="/seen/{{ID}}" method="POST">
						<button type="submit" class="btn btn-xs btn-success">
							<span class="glyphicon glyphicon-ok"></span>
						</button>
					</form>
					<form style="display: inline" action="/delete/{{ID}}" method="POST">
						<button type="submit" class="btn btn-xs btn-danger" onclick="return confirmDelete();">
							<span class="glyphicon glyphicon-remove"></span>
						</button>
					</form>
					<form style="display: inline" action="/update/{{ID}}" method="POST">
						<button type="submit" class="btn btn-xs btn-primary">
							<span class="glyphicon glyphicon-refresh"></span>
						</button>
					</form>
				</td>
			</tr>
		{{/items}}
	</table>

	<form class="form-inline" action="/addnew" method="POST">
		<input type="text" class="form-control"
			id="url" name="url" placeholder="URL to check">
		<input type="text" class="form-control"
			id="selector" name="selector" placeholder="Selector to monitor">
		<input type="text" class="form-control"
			id="schedule" name="schedule" placeholder="Schedule">
		<button type="submit" class="btn btn-primary">Add</button>
		<button type="submit" class="btn btn-default">Clear</button>
	</form>

	<script>
		$(document).ready(function() {
			$('[data-hash]').each(function(el) {
				var $this = $(this);
				var fullHash = $this.attr('data-hash');
				var toDisplay = "<input type='text' value='" + fullHash + "' readonly />";

				$this.popover({
					html:		true,
					placement:	'top',
					trigger:	'click',
					content:	toDisplay,
				});
			});
		});

		var confirmDelete = function() {
			return confirm("Do you really want to delete this check?");
		};
	</script>
	`)
}
