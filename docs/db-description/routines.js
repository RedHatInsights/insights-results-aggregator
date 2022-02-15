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

var filterBy = function(functionType) {
    $.fn.dataTableExt.afnFiltering.length = 0;
    $.fn.dataTable.ext.search.push(
        function( settings, data, dataIndex ) {
            var type = data[1]; // use data for the Type column

            if ( type.toUpperCase() == functionType || functionType == 'All' )
            {
                return true;
            }
            return false;
        }
    );
}

$(document).ready(function() {
	var activeObject;
    var table = $('#routine_table').DataTable( {
        lengthChange: false,
		ordering: true,
		paging: config.pagination,
		pageLength: 50,
		autoWidth: true,
		processing: true,
		order: [[ 0, "asc" ]],
        buttons: [
            {
                text: 'All',
                action: function ( e, dt, node, config ) {
                    filterBy('All');
                    if (activeObject != null) {
                        activeObject.active(false);
                    }
                    table.draw();
                }
            },
            {
                text: 'Functions',
                action: function ( e, dt, node, config ) {
                    filterBy('FUNCTION');
                    if (activeObject != null) {
                        activeObject.active(false);
                    }
                    this.active( !this.active() );
                    activeObject = this;
                    table.draw();
                }
            },
            {
                text: 'Procedures',
                action: function ( e, dt, node, config ) {
                    filterBy('PROCEDURE');
                    if (activeObject != null) {
                        activeObject.active(false);
                    }
                    this.active( !this.active() );
                    activeObject = this;
                    table.draw();
                }
            },
            {
                extend: 'columnsToggle',
                columns: '.toggle'
            }
        ]
    } );

    //schemaSpy.js
    dataTableExportButtons(table);
} );
