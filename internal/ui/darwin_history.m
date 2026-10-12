//go:build darwin && cgo
#import "darwin_desktop.h"

static NSString *HMasked=@"••••••••••••••••••••";

@implementation CPDesktop (History)

-(NSStackView *)buildHistory {
 self.historyData=[NSMutableArray array];
 NSButton *back=CPIcon(@"chevron.left",@"Вернуться к устройствам",self,31);
 NSTextField *title=CPLabel(@"История",16);
 self.emptyHistory=CPLabel(@"История пуста\nСкопированные и полученные элементы появятся здесь автоматически.",14);
 self.emptyHistory.textColor=NSColor.secondaryLabelColor;
 NSScrollView *scroll=[[[NSScrollView alloc]init]autorelease];scroll.hasVerticalScroller=YES;scroll.drawsBackground=NO;scroll.borderType=NSNoBorder;scroll.translatesAutoresizingMaskIntoConstraints=NO;
 NSLayoutConstraint *height=[scroll.heightAnchor constraintEqualToConstant:280];height.identifier=@"historyListHeight";height.active=YES;
 NSTableView *table=[[[NSTableView alloc]initWithFrame:NSMakeRect(0,0,480,280)]autorelease];NSTableColumn *column=[[[NSTableColumn alloc]initWithIdentifier:@"history"]autorelease];column.width=480;column.editable=NO;column.resizingMask=NSTableColumnAutoresizingMask;[table addTableColumn:column];table.headerView=nil;table.rowHeight=76;table.dataSource=self;table.delegate=self;table.backgroundColor=NSColor.clearColor;table.autoresizingMask=NSViewWidthSizable;scroll.documentView=table;
 if(@available(macOS 11.0,*))table.style=NSTableViewStyleFullWidth;
 self.historyTable=table;self.historyList=scroll;self.historyList.hidden=YES;
 return CPStack(@[CPStack(@[back,title],YES),self.emptyHistory,self.historyList],NO);
}

-(NSInteger)historyRowCount {return self.historyData.count;}

// One shared tableView: implementation lives in the Home category; History
// contributes only its own cell builder to avoid duplicate selectors.
-(NSView *)historyCellForColumn:(NSTableColumn *)column row:(NSInteger)row {
 NSDictionary *item=self.historyData[row];
 NSTableCellView *cell=[[[NSTableCellView alloc]initWithFrame:NSMakeRect(0,0,column.width,76)]autorelease];
 if([item[@"group"]boolValue]){
  NSTextField *label=[NSTextField labelWithString:item[@"header"]?:@""];label.font=[NSFont systemFontOfSize:12];label.textColor=NSColor.secondaryLabelColor;label.frame=NSMakeRect(4,28,column.width-8,20);label.autoresizingMask=NSViewWidthSizable;[cell addSubview:label];
  return cell;
 }
 NSTextField *header=[NSTextField labelWithString:item[@"header"]?:@""];header.font=[NSFont systemFontOfSize:12];header.textColor=NSColor.secondaryLabelColor;header.lineBreakMode=NSLineBreakByTruncatingTail;header.frame=NSMakeRect(4,54,column.width-8,18);header.autoresizingMask=NSViewWidthSizable;[cell addSubview:header];
 NSTextField *preview=[NSTextField labelWithString:item[@"preview"]?:@""];preview.font=[NSFont systemFontOfSize:13];preview.lineBreakMode=NSLineBreakByTruncatingTail;preview.maximumNumberOfLines=3;preview.frame=NSMakeRect(4,6,column.width-186,48);preview.autoresizingMask=NSViewWidthSizable;[cell addSubview:preview];
 if([item[@"url"]boolValue])preview.textColor=NSColor.controlAccentColor;
 NSButton *trash=CPIcon(@"trash",@"Удалить запись",self,36);trash.action=@selector(historyAction:);[cell addSubview:trash];
 NSButton *copy=CPIcon(@"doc.on.doc",@"Копировать",self,34);copy.action=@selector(historyAction:);[cell addSubview:copy];
 if([item[@"url"]boolValue]){
  NSButton *link=CPIcon(@"safari",@"Открыть ссылку",self,0);link.action=@selector(historyLink:);[cell addSubview:link];[NSLayoutConstraint activateConstraints:@[[link.trailingAnchor constraintEqualToAnchor:cell.trailingAnchor constant:-8],[link.centerYAnchor constraintEqualToAnchor:cell.centerYAnchor constant:-30]]];
 }
 [NSLayoutConstraint activateConstraints:@[[trash.trailingAnchor constraintEqualToAnchor:cell.trailingAnchor constant:-8],[trash.centerYAnchor constraintEqualToAnchor:cell.centerYAnchor constant:30],[copy.trailingAnchor constraintEqualToAnchor:cell.trailingAnchor constant:-8],[copy.centerYAnchor constraintEqualToAnchor:cell.centerYAnchor]]];
 if([item[@"sensitive"]boolValue]){
  NSButton *eye=CPIcon([item[@"revealed"]boolValue]?@"eye.slash":@"eye",[item[@"revealed"]boolValue]?@"Скрыть значение":@"Показать значение",self,35);eye.action=@selector(historyAction:);[cell addSubview:eye];
  [NSLayoutConstraint activateConstraints:@[[eye.trailingAnchor constraintEqualToAnchor:copy.leadingAnchor constant:-6],[eye.centerYAnchor constraintEqualToAnchor:cell.centerYAnchor]]];
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

-(void)refreshHistory {
 for(NSLayoutConstraint *constraint in self.historyList.constraints)if([constraint.identifier isEqualToString:@"historyListHeight"])constraint.constant=MIN(MAX(self.historyData.count*76+4,64),320);
 self.historyList.hidden=self.historyData.count==0;
 self.emptyHistory.hidden=self.historyData.count>0;
 [self.historyTable reloadData];
 [self resizeDocument];
}

@end
