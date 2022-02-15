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

$(function() {
     var pgurl = window.location.href.substr(window.location.href.lastIndexOf("/")+1);
     $("#navbar-collapse ul li a").each(function(){
          if($(this).attr("href") == pgurl || $(this).attr("href") == '' )
          $(this).parent().addClass("active");
     })
});

function dataTableExportButtons(table) {
    $("<div class=\"row\">\n" +
        "    <div id=\"button_group_one\" class=\"col-md-6 col-sm-6\"></div>\n" +
        "    <div id=\"button_group_two\" class=\"col-md-2 col-sm-4 pull-right text-right\"></div>\n" +
        "</div>").prependTo('#' + table.table().container().id);
   new $.fn.dataTable.Buttons( table, {
       name: 'exports',
       buttons: [
           {
               extend:    'copyHtml5',
               text:      '<i class="fa fa-files-o"></i>',
               titleAttr: 'Copy'
           },
           {
               extend:    'excelHtml5',
               text:      '<i class="fa fa-file-excel-o"></i>',
               titleAttr: 'Excel'
           },
           {
               extend:    'csvHtml5',
               text:      '<i class="fa fa-file-text-o"></i>',
               titleAttr: 'CSV'
           },
           {
               extend:    'pdfHtml5',
               text:      '<i class="fa fa-file-pdf-o"></i>',
               orientation: 'landscape',
               titleAttr: 'PDF'
           }
       ]
   } );

    table.buttons().container().appendTo( '#' + table.table().container().id + ' #button_group_one' );
    table.buttons( 'exports', null ).container().appendTo( '#' + table.table().container().id + ' #button_group_two' );
}

 