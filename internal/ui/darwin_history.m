//go:build darwin && cgo
#import "darwin_desktop.h"

// Forward wheel events to the document scroll when the history list fits or
// reaches its edges, so the whole window keeps scrolling over the block.
@interface CPForwardScroll : NSScrollView
@end
@implementation CPForwardScroll
-(void)scrollWheel:(NSEvent *)event {
 NSView *document=self.documentView;
 CGFloat contentHeight=document?document.frame.size.height:0;
 BOOL canScroll=contentHeight>self.contentSize.height+1;
 CGFloat delta=event.scrollingDeltaY;
 BOOL atTop=NO,atBottom=NO;
 if(canScroll){
  CGFloat visibleTop=self.contentView.bounds.origin.y;
  CGFloat visibleBottom=visibleTop+self.contentSize.height;
  atTop=visibleTop<=0.5;atBottom=visibleBottom>=contentHeight-0.5;
 }
 // Nested scrolling: only hand the wheel to the window at the edges —
 // scrolling up past the top or down past the bottom.
 if(!canScroll||(delta>0&&atTop)||(delta<0&&atBottom)){
  [[self nextResponder]tryToPerform:@selector(scrollWheel:) with:event];return;
 }
 [super scrollWheel:event];
}
@end

static const NSInteger HHeaderTag=1,HPreviewTag=2;

@implementation CPDesktop (History)

-(NSStackView *)buildHistory {
 self.historyData=[NSMutableArray array];self.historyIds=[NSMutableArray array];self.historyHeights=[NSMutableArray array];
 NSButton *back=CPIcon(@"chevron.left",@"Вернуться к устройствам",self,31);
 NSTextField *title=CPLabel(@"История",16);
 self.emptyHistory=CPLabel(@"История пуста\nСкопированные и полученные элементы появятся здесь автоматически.",14);
 self.emptyHistory.textColor=NSColor.secondaryLabelColor;
 CPForwardScroll *scroll=[[[CPForwardScroll alloc]init]autorelease];scroll.hasVerticalScroller=YES;scroll.drawsBackground=NO;scroll.borderType=NSNoBorder;scroll.translatesAutoresizingMaskIntoConstraints=NO;
 NSLayoutConstraint *height=[scroll.heightAnchor constraintEqualToConstant:280];height.identifier=@"historyListHeight";height.active=YES;
 NSTableView *table=[[[NSTableView alloc]initWithFrame:NSMakeRect(0,0,480,280)]autorelease];NSTableColumn *column=[[[NSTableColumn alloc]initWithIdentifier:@"history"]autorelease];column.width=480;column.editable=NO;column.resizingMask=NSTableColumnAutoresizingMask;[table addTableColumn:column];table.headerView=nil;table.rowHeight=64;table.dataSource=self;table.delegate=self;table.backgroundColor=NSColor.clearColor;table.autoresizingMask=NSViewWidthSizable;scroll.documentView=table;
 if(@available(macOS 11.0,*))table.style=NSTableViewStyleFullWidth;
 self.historyTable=table;self.historyList=scroll;self.historyList.hidden=YES;
 return CPStack(@[CPStack(@[back,title],YES),self.emptyHistory,self.historyList],NO);
}

// Shared delegate entry points live in the Home category; History only adds
// variable row heights and its own cell builder.
-(CGFloat)tableView:(NSTableView *)tableView heightOfRow:(NSInteger)row {
 if(tableView!=self.historyTable)return 52;
 if(row<0||row>=(NSInteger)self.historyData.count)return 64;
 return MAX(28,MIN([self.historyData[row][@"height"]doubleValue],600));
}

-(NSInteger)historyRowCount {return self.historyData.count;}

-(NSView *)historyCellForColumn:(NSTableColumn *)column row:(NSInteger)row {
 NSDictionary *item=self.historyData[row];
 CGFloat cellHeight=MAX(28,MIN([item[@"height"]doubleValue],600));
 NSTableCellView *cell=[[[NSTableCellView alloc]initWithFrame:NSMakeRect(0,0,column.width,cellHeight)]autorelease];
 if([item[@"group"]boolValue]){
  NSTextField *label=[NSTextField labelWithString:item[@"header"]?:@""];label.font=[NSFont systemFontOfSize:12];label.textColor=NSColor.secondaryLabelColor;label.tag=HHeaderTag;label.frame=NSMakeRect(4,0,column.width-8,cellHeight);label.autoresizingMask=NSViewWidthSizable|NSViewHeightSizable;[cell addSubview:label];
  return cell;
 }
 BOOL expanded=[item[@"expanded"]boolValue];
 // Header on top, preview fills the rest; buttons align with the header line.
 NSTextField *header=[NSTextField labelWithString:item[@"header"]?:@""];header.font=[NSFont systemFontOfSize:12];header.textColor=NSColor.secondaryLabelColor;header.lineBreakMode=NSLineBreakByTruncatingTail;header.tag=HHeaderTag;header.frame=NSMakeRect(4,cellHeight-26,column.width-8,18);header.autoresizingMask=NSViewWidthSizable|NSViewMinYMargin;[cell addSubview:header];
 NSTextField *preview=[NSTextField labelWithString:item[@"preview"]?:@""];preview.font=[NSFont systemFontOfSize:13];preview.tag=HPreviewTag;preview.frame=NSMakeRect(4,4,column.width-198,cellHeight-32);preview.autoresizingMask=NSViewWidthSizable|NSViewHeightSizable;
 if(expanded){preview.lineBreakMode=NSLineBreakByWordWrapping;preview.maximumNumberOfLines=0;}
 else{preview.lineBreakMode=NSLineBreakByTruncatingTail;preview.maximumNumberOfLines=2;}
 if([item[@"url"]boolValue])preview.textColor=NSColor.controlAccentColor;
 [cell addSubview:preview];
 // All actions share one row on the header line, right-aligned:
 // copy sits left of delete.
 NSButton *copy=CPIcon(@"doc.on.doc",@"Копировать",self,34);copy.action=@selector(historyAction:);[cell addSubview:copy];
 NSButton *trash=CPIcon(@"trash",@"Удалить запись",self,36);trash.action=@selector(historyAction:);[cell addSubview:trash];
 NSButton *last=trash;
 [NSLayoutConstraint activateConstraints:@[[trash.trailingAnchor constraintEqualToAnchor:cell.trailingAnchor constant:-8],[trash.centerYAnchor constraintEqualToAnchor:header.centerYAnchor],[copy.trailingAnchor constraintEqualToAnchor:trash.leadingAnchor constant:-6],[copy.centerYAnchor constraintEqualToAnchor:header.centerYAnchor]]];
 last=copy;
 if([item[@"sensitive"]boolValue]){
  NSButton *eye=CPIcon([item[@"revealed"]boolValue]?@"eye.slash":@"eye",[item[@"revealed"]boolValue]?@"Скрыть значение":@"Показать значение",self,35);eye.action=@selector(historyAction:);[cell addSubview:eye];
  [NSLayoutConstraint activateConstraints:@[[eye.trailingAnchor constraintEqualToAnchor:last.leadingAnchor constant:-6],[eye.centerYAnchor constraintEqualToAnchor:header.centerYAnchor]]];
  last=eye;
 }
 if([item[@"url"]boolValue]){
  NSButton *link=CPIcon(@"safari",@"Открыть ссылку",self,0);link.action=@selector(historyLink:);[cell addSubview:link];
  [NSLayoutConstraint activateConstraints:@[[link.trailingAnchor constraintEqualToAnchor:last.leadingAnchor constant:-6],[link.centerYAnchor constraintEqualToAnchor:header.centerYAnchor]]];
  last=link;
 }
 if([item[@"truncated"]boolValue]){
  NSButton *expand=CPIcon([item[@"expanded"]boolValue]?@"chevron.down":@"chevron.right",[item[@"expanded"]boolValue]?@"Свернуть запись":@"Развернуть запись",self,45);expand.action=@selector(historyAction:);[cell addSubview:expand];
  [NSLayoutConstraint activateConstraints:@[[expand.trailingAnchor constraintEqualToAnchor:last.leadingAnchor constant:-6],[expand.centerYAnchor constraintEqualToAnchor:header.centerYAnchor]]];
 }
 return cell;
}

-(void)selectHistoryRow:(NSButton *)sender {
 NSInteger row=[self.historyTable rowForView:sender];
 if(row>=0&&row<(NSInteger)self.historyData.count)[self.historyTable selectRowIndexes:[NSIndexSet indexSetWithIndex:row] byExtendingSelection:NO];
}

-(void)historyAction:(NSButton *)sender {
 if(self.window.attachedSheet)return;
 [self selectHistoryRow:sender];
 [self.events addObject:@(sender.tag)];
}

-(void)historyLink:(NSButton *)sender {
 NSInteger row=[self.historyTable rowForView:sender];
 if(row<0||row>=(NSInteger)self.historyData.count)return;
 NSString *link=self.historyData[row][@"link"];
 if(![link isKindOfClass:NSString.class])return;
 if(![link hasPrefix:@"http://"]&&![link hasPrefix:@"https://"])return;
 NSURL *url=[NSURL URLWithString:link];
 if(url)[[NSWorkspace sharedWorkspace]openURL:url];
}

// refreshHistory avoids full reloads when the row set is unchanged: repeated
// background refreshes and button clicks must not blink the list.
-(void)refreshHistory {
 CGFloat total=0;for(NSDictionary *item in self.historyData)total+=MAX(28,MIN([item[@"height"]doubleValue],600));
 for(NSLayoutConstraint *constraint in self.historyList.constraints)if([constraint.identifier isEqualToString:@"historyListHeight"])constraint.constant=MIN(MAX(total+4,64),320);
 self.historyList.hidden=self.historyData.count==0;
 self.emptyHistory.hidden=self.historyData.count>0;
 NSMutableArray *ids=[NSMutableArray array],*heights=[NSMutableArray array];
 for(NSDictionary *item in self.historyData){
  [ids addObject:item[@"id"]?:@""];
  [heights addObject:@(MAX(28,MIN([item[@"height"]doubleValue],600)))];
 }
 BOOL canUpdateInPlace=self.historyTable.numberOfRows==(NSInteger)self.historyData.count
  &&[ids isEqualToArray:self.historyIds]&&[heights isEqualToArray:self.historyHeights];
 self.historyIds=ids;self.historyHeights=heights;
 if(canUpdateInPlace){
  for(NSInteger row=0;row<(NSInteger)self.historyData.count;row++){
   NSDictionary *item=self.historyData[row];
   NSView *cell=[self.historyTable viewAtColumn:0 row:row makeIfNecessary:NO];
   if(!cell)continue;
   NSTextField *header=(NSTextField *)[cell viewWithTag:HHeaderTag];
   NSTextField *preview=(NSTextField *)[cell viewWithTag:HPreviewTag];
   if(header)header.stringValue=item[@"header"]?:@"";
   if(preview)preview.stringValue=item[@"preview"]?:@"";
   for(NSView *subview in cell.subviews){
    if(![subview isKindOfClass:NSButton.class])continue;
    NSButton *button=(NSButton *)subview;
    if(button.tag==35)button.image=CPSymbol([item[@"revealed"]boolValue]?@"eye.slash":@"eye",nil);
    if(button.tag==45)button.image=CPSymbol([item[@"expanded"]boolValue]?@"chevron.down":@"chevron.right",nil);
   }
  }
 }else{
  [self.historyTable reloadData];
 }
 [self resizeDocument];
}

@end
