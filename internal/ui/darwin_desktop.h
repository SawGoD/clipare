#import <AppKit/AppKit.h>
#import <UserNotifications/UserNotifications.h>

typedef NS_ENUM(NSInteger, CPView) { CPHome, CPDiscovery, CPPairing, CPUpdate, CPNotice };
@interface CPStatusDot : NSView
@property NSInteger state;
@end
@interface CPDesktop : NSObject <NSWindowDelegate>
@property(retain) NSWindow *window;
@property(retain) NSScrollView *scroll;
@property(retain) NSMutableDictionary *views;
@property(retain) NSMutableArray *fields, *events, *peers, *found;
@property(retain) NSStatusItem *tray;
@property(retain) CPStatusDot *statusDot;
@property(retain) NSButton *autostart,*updates,*additionalButton,*advancedButton,*pause,*emptyAdd,*compactAdd,*removePeer,*connect,*approve,*reject,*install,*dismiss;
@property(retain) NSStackView *additional,*advanced;
@property(retain) NSTableView *peerTable,*foundTable;
@property(retain) NSView *emptyDevices,*peerList,*foundList,*emptyFound;
@property(retain) NSTextField *status,*reason,*version,*discoveryStatus,*pairName,*sas,*pairHelp,*updateText,*noticeText;
@property(retain) NSString *statusMessage,*statusReason;
@property NSInteger view, syncState;
@property BOOL pairIncoming, additionalExpanded, advancedExpanded;
@property BOOL devicesEmpty, canRemove, compactVisible, listVisible;
@property BOOL closed;
-(void)build;
-(void)action:(id)sender;
-(void)show:(NSInteger)view;
-(void)back;
-(void)resizeDocument;
-(NSView *)document:(NSStackView *)stack;
-(void)installEditMenu;
@end

@interface CPDesktop (Home) <NSTableViewDataSource,NSTableViewDelegate>
-(void)refreshDevices;
-(NSStackView *)buildHome;
-(NSStackView *)buildAdvanced;
-(NSView *)table:(BOOL)discovery;
@end
@interface CPDesktop (Flows)
-(void)buildFlows;
@end
@interface CPDesktop (Status)
-(void)refreshTray;
@end
@interface CPDesktop (Notifications) <UNUserNotificationCenterDelegate>
-(void)prepareNotifications;
-(void)notifyPair:(NSString *)name;
-(void)clearPairNotification;
@end

NSTextField *CPLabel(NSString *text,CGFloat size);
NSStackView *CPStack(NSArray *views,BOOL horizontal);
void CPAdd(NSStackView *stack,NSView *view);
NSView *CPCard(NSStackView *stack);
NSButton *CPButton(NSString *title,CPDesktop *desktop,NSInteger tag);
NSButton *CPIcon(NSString *symbol,NSString *label,CPDesktop *desktop,NSInteger tag);
NSTextField *CPInput(CPDesktop *desktop,NSInteger index,BOOL secure);
NSColor *CPStatusColor(NSInteger state);
NSImage *CPSymbol(NSString *symbol,NSString *label);
