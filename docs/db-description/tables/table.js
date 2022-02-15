/**
 * Copyright 2022 Red Hat, Inc
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

$(document).ready(function() {
    anchors.options.visible = 'always';
    anchors.add('h3');

    var table = $('#standard_table').DataTable( {
        lengthChange: false,
        ordering: false,
        paging: config.pagination,
		autoWidth: true,
		buttons: [
					{
						text: 'Related columns',
						action: function ( e, dt, node, config ) {
							$(".relatedKey").toggle();
							this.active( !this.active() );
							table.columns.adjust().draw();
						}
					},
					{
						text: 'Constraint',
						action: function ( e, dt, node, config ) {
							$(".constraint").toggle();
							this.active( !this.active() );
							table.columns.adjust().draw();
						}
					},
					{
						extend: 'columnsToggle',
						columns: '.toggle'
					}
				]

    } );
    dataTableExportButtons(table);

    if ($('#indexes_table').length) {
        var indexes = $('#indexes_table').DataTable({
            lengthChange: false,
            paging: config.pagination,
            ordering: false
        });
        dataTableExportButtons(indexes);
    }

    if ($('#check_table').length) {
        var check = $('#check_table').DataTable( {
            lengthChange: false,
            paging: config.pagination,
            ordering: false
        } );
        dataTableExportButtons(check);
    }
} );


$(function() {
	var $imgs = $('img.diagram, object.diagram');
	$imgs.css("cursor", "move")
	$imgs.draggable();
});

$.fn.digits = function(){
	return this.each(function(){
		$(this).text( $(this).text().replace(/(\d)(?=(\d\d\d)+(?!\d))/g, "1 ") );
	})
}

$(function() {
	$("#recordNumber").digits();
});

var codeElement = document.getElementById("sql-script-codemirror");
var editor = null;
if (null != codeElement) {
	editor = CodeMirror.fromTextArea(codeElement, {
		lineNumbers: true,
		mode: 'text/x-sql',
		indentWithTabs: true,
		smartIndent: true,
		lineNumbers: true,
		matchBrackets: true,
		autofocus: true,
        readOnly: true
	});
}
